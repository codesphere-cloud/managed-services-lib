// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/codesphere-cloud/managed-services-lib/model"
)

// Defaults applied to a JobSpec when the corresponding field is left zero. A
// backup job runs detached and must not run forever, so a bounded deadline is
// always set; a modest TTL keeps a finished Job around long enough for a status
// poll to observe the outcome before garbage collection.
const (
	DefaultJobTimeout      = 6 * time.Hour
	DefaultJobTTL          = 1 * time.Hour
	DefaultJobBackoffLimit = int32(3)

	// jobContainerName is the single container every dispatched Job runs. State
	// reads logs from this container to explain failures.
	jobContainerName = "job"

	// failureLogTailLines bounds how much of a failed pod's log is folded into
	// the reported failure reason.
	failureLogTailLines int64 = 20
)

// JobPhase is the coarse lifecycle state of a dispatched Job, derived from its
// status. It maps cleanly onto higher-level phase models (e.g. a backup phase).
type JobPhase string

const (
	// JobNotFound means no Job of that name exists — never dispatched, or already
	// garbage-collected after its TTL.
	JobNotFound JobPhase = "not_found"
	// JobPending means the Job exists but no pod is running yet.
	JobPending JobPhase = "pending"
	// JobRunning means at least one pod is active.
	JobRunning JobPhase = "running"
	// JobSucceeded means the Job completed successfully.
	JobSucceeded JobPhase = "succeeded"
	// JobFailed means the Job exhausted its retries or hit its deadline.
	JobFailed JobPhase = "failed"
)

// JobState is a point-in-time snapshot of a Job, shaped for a polling caller.
type JobState struct {
	// Phase is the coarse lifecycle state.
	Phase JobPhase
	// StartedAt is when the Job first started a pod, if it has.
	StartedAt *time.Time
	// FinishedAt is when the Job reached a terminal phase, if it has.
	FinishedAt *time.Time
	// Reason explains a JobFailed phase: the failure condition message enriched,
	// when available, with the tail of the failed pod's logs. Empty otherwise.
	Reason string
}

// JobSpec describes a one-shot Job to dispatch. Name is both the Job (and
// credentials Secret) name and the idempotency key: re-dispatching while a Job
// of the same name exists is a no-op rather than a duplicate.
type JobSpec struct {
	// Name is the Job/Secret name and idempotency key.
	Name string
	// Image is the container image the Job runs.
	Image string
	// Command overrides the image entrypoint.
	Command []string
	// Args are the container arguments.
	Args []string
	// Env holds plain (non-secret) environment variables.
	Env map[string]string
	// Secrets holds secret environment variables. Each entry is stored in the
	// Job's owned Secret and injected via a secretKeyRef, so values never appear
	// in the Job or Pod manifest. The map keys are the env var names — there is
	// no separate key list to keep in sync.
	Secrets map[string]string
	// Labels are applied to the Job and its Secret. Providers should set the
	// standard service labels so the resources are discoverable and cleaned up
	// with the service.
	Labels map[string]string

	// Timeout bounds the Job's total run time (activeDeadlineSeconds).
	// Zero means DefaultJobTimeout.
	Timeout time.Duration
	// TTL is how long a finished Job is kept before garbage collection
	// (ttlSecondsAfterFinished). Zero means DefaultJobTTL.
	TTL time.Duration
	// BackoffLimit bounds pod retries. Nil means DefaultJobBackoffLimit.
	BackoffLimit *int32
	// Resources sets the container resource requests/limits.
	Resources corev1.ResourceRequirements

	// Customize, if set, is applied to the fully-built Job before it is created.
	// It is the escape hatch for the rare field this struct does not model.
	Customize func(*batchv1.Job)
}

// JobRunner dispatches and inspects one-shot Kubernetes Jobs.
// It is a composed helper written against the KubernetesClient
// primitives, so it stays unit-testable against a mocked client and is reusable
// for any detached work (backups, restores, migrations).
type JobRunner struct {
	kube KubernetesClient
}

// NewJobRunner returns a JobRunner backed by the given client.
func NewJobRunner(kube KubernetesClient) JobRunner {
	return JobRunner{kube: kube}
}

// Run dispatches the Job described by spec into namespace. It is idempotent: if
// a Job of the same name already exists the call is a no-op. When spec.Secrets
// is non-empty a credentials Secret is created first (so the pod never starts
// without it) and then adopted by the Job for garbage collection.
func (r JobRunner) Run(ctx context.Context, namespace string, spec JobSpec) error {
	hasSecret := len(spec.Secrets) > 0
	if hasSecret {
		if _, err := r.kube.CreateSecret(ctx, namespace, buildJobSecret(spec, namespace)); err != nil &&
			!errors.Is(err, ErrResourceConflict) {
			return fmt.Errorf("creating credentials secret for job %s: %w", spec.Name, err)
		}
	}

	job, err := buildJob(spec, namespace)
	if err != nil {
		return fmt.Errorf("building job %s: %w", spec.Name, err)
	}

	created, err := r.kube.Create(ctx, model.JobAPI, namespace, job)
	if err != nil {
		if errors.Is(err, ErrResourceConflict) {
			// Job already dispatched; its Secret already exists too. No-op.
			return nil
		}
		return fmt.Errorf("creating job %s: %w", spec.Name, err)
	}

	if hasSecret {
		// Best-effort: adopt the Secret so it is garbage-collected with the Job.
		// If this fails the Secret is still removed explicitly by Delete.
		r.adoptSecret(ctx, namespace, spec.Name, created.GetUID())
	}
	return nil
}

// State returns a point-in-time snapshot of the named Job. A Job that does not
// exist is reported as JobNotFound with no error. A failed Job's Reason is
// enriched with the tail of the failed pod's logs when they can be read.
func (r JobRunner) State(ctx context.Context, namespace, name string) (JobState, error) {
	obj, err := r.kube.Get(ctx, model.ResourceRef{APIResource: model.JobAPI, Namespace: namespace, Name: name})
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			return JobState{Phase: JobNotFound}, nil
		}
		return JobState{}, fmt.Errorf("getting job %s: %w", name, err)
	}

	var job batchv1.Job
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &job); err != nil {
		return JobState{}, fmt.Errorf("decoding job %s: %w", name, err)
	}

	state := JobState{
		Phase:      JobPending,
		StartedAt:  timePtr(job.Status.StartTime),
		FinishedAt: timePtr(job.Status.CompletionTime),
	}
	switch {
	case jobConditionTrue(job, batchv1.JobComplete):
		state.Phase = JobSucceeded
	case jobConditionTrue(job, batchv1.JobFailed):
		state.Phase = JobFailed
		state.Reason = r.failureReason(ctx, job)
		if state.FinishedAt == nil {
			state.FinishedAt = conditionTime(job, batchv1.JobFailed)
		}
	case job.Status.Active > 0:
		state.Phase = JobRunning
	}
	return state, nil
}

// Delete removes the named Job and its credentials Secret. Resources that no
// longer exist are not an error, so Delete is safe to call unconditionally.
func (r JobRunner) Delete(ctx context.Context, namespace, name string) error {
	if err := r.kube.Delete(ctx, model.ResourceRef{APIResource: model.JobAPI, Namespace: namespace, Name: name}); err != nil &&
		!errors.Is(err, ErrResourceNotFound) {
		return fmt.Errorf("deleting job %s: %w", name, err)
	}
	// The Secret is normally garbage-collected via its owner reference; delete it
	// explicitly too in case adoption did not happen.
	if err := r.kube.DeleteSecret(ctx, namespace, name); err != nil &&
		!errors.Is(err, ErrResourceNotFound) {
		return fmt.Errorf("deleting credentials secret for job %s: %w", name, err)
	}
	return nil
}

// Replace deletes any existing Job of the same name and dispatches a fresh one.
// It is the retry path: a failed Job must be removed before it can be re-run,
// because Run is a no-op while a Job of the same name exists. If the previous
// Job is still terminating the re-dispatch no-ops; call again on the next poll.
func (r JobRunner) Replace(ctx context.Context, namespace string, spec JobSpec) error {
	if err := r.Delete(ctx, namespace, spec.Name); err != nil {
		return err
	}
	return r.Run(ctx, namespace, spec)
}

// adoptSecret patches the Job's owner reference onto its credentials Secret so
// the Secret is garbage-collected together with the Job.
// Best-effort: failures are tolerated because Delete removes the Secret explicitly as a fallback.
func (r JobRunner) adoptSecret(ctx context.Context, namespace, name string, jobUID types.UID) {
	owner := []metav1.OwnerReference{{
		APIVersion: "batch/v1",
		Kind:       "Job",
		Name:       name,
		UID:        jobUID,
	}}
	_ = r.kube.Patch(ctx, model.ResourceRef{APIResource: model.SecretAPI, Namespace: namespace, Name: name},
		[]model.K8sPatch{{Op: "add", Path: "/metadata/ownerReferences", Value: owner}})
}

// failureReason returns the Job's failure condition message, enriched with the
// tail of the failed pod's logs when they can be read. The condition alone
// ("BackoffLimitExceeded") rarely explains what went wrong; the logs do.
func (r JobRunner) failureReason(ctx context.Context, job batchv1.Job) string {
	reason := conditionMessage(job, batchv1.JobFailed)
	logs := r.failedPodLogs(ctx, job.Namespace, job.Name)
	if logs == "" {
		return reason
	}
	if reason == "" {
		return logs
	}
	return reason + ": " + logs
}

// failedPodLogs returns the tail of the logs of a pod belonging to the Job.
// Best-effort: any error yields an empty string.
func (r JobRunner) failedPodLogs(ctx context.Context, namespace, jobName string) string {
	pods, err := r.kube.List(ctx, model.ListOptions{
		APIResource:   model.PodAPI,
		Namespace:     namespace,
		LabelSelector: "job-name=" + jobName,
	})
	if err != nil || pods == nil || len(pods.Items) == 0 {
		return ""
	}
	// The most recently created pod is the one that exhausted the retries.
	podName := pods.Items[len(pods.Items)-1].GetName()
	logs, err := r.kube.GetPodLogs(ctx, namespace, podName, jobContainerName, failureLogTailLines)
	if err != nil {
		return ""
	}
	return logs
}

func buildJob(spec JobSpec, namespace string) (*unstructured.Unstructured, error) {
	ttl := int32(durationSeconds(spec.TTL, DefaultJobTTL))
	deadline := durationSeconds(spec.Timeout, DefaultJobTimeout)
	backoff := DefaultJobBackoffLimit
	if spec.BackoffLimit != nil {
		backoff = *spec.BackoffLimit
	}

	job := &batchv1.Job{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: namespace, Labels: spec.Labels},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttl,
			ActiveDeadlineSeconds:   &deadline,
			BackoffLimit:            &backoff,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: spec.Labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{{
						Name:            jobContainerName,
						Image:           spec.Image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         spec.Command,
						Args:            spec.Args,
						Env:             buildEnv(spec),
						Resources:       spec.Resources,
					}},
				},
			},
		},
	}
	if spec.Customize != nil {
		spec.Customize(job)
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		return nil, fmt.Errorf("converting job to unstructured: %w", err)
	}
	return &unstructured.Unstructured{Object: m}, nil
}

func buildJobSecret(spec JobSpec, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: namespace, Labels: spec.Labels},
		Type:       corev1.SecretTypeOpaque,
		StringData: spec.Secrets,
	}
}

// buildEnv builds the container env: plain vars from Env, then secret vars from
// Secrets routed through a secretKeyRef against the Job's own Secret. Both are
// emitted in sorted key order so the produced Job is deterministic.
func buildEnv(spec JobSpec) []corev1.EnvVar {
	out := make([]corev1.EnvVar, 0, len(spec.Env)+len(spec.Secrets))
	for _, k := range sortedKeys(spec.Env) {
		out = append(out, corev1.EnvVar{Name: k, Value: spec.Env[k]})
	}
	for _, k := range sortedKeys(spec.Secrets) {
		out = append(out, corev1.EnvVar{
			Name: k,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: spec.Name},
					Key:                  k,
				},
			},
		})
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func durationSeconds(d, fallback time.Duration) int64 {
	if d <= 0 {
		d = fallback
	}
	return int64(d.Seconds())
}

func jobConditionTrue(job batchv1.Job, t batchv1.JobConditionType) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == t {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func conditionMessage(job batchv1.Job, t batchv1.JobConditionType) string {
	for _, c := range job.Status.Conditions {
		if c.Type != t {
			continue
		}
		if c.Message != "" {
			return c.Message
		}
		if c.Reason != "" {
			return c.Reason
		}
	}
	return "unknown reason"
}

func conditionTime(job batchv1.Job, t batchv1.JobConditionType) *time.Time {
	for _, c := range job.Status.Conditions {
		if c.Type == t && !c.LastTransitionTime.IsZero() {
			return timePtr(&c.LastTransitionTime)
		}
	}
	return nil
}

func timePtr(t *metav1.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	out := t.Time
	return &out
}
