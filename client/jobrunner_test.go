// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/codesphere-cloud/managed-services-lib/client"
	"github.com/codesphere-cloud/managed-services-lib/client/mocks"
	"github.com/codesphere-cloud/managed-services-lib/model"
)

func toUnstructured(obj interface{}) *unstructured.Unstructured {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	Expect(err).NotTo(HaveOccurred())
	return &unstructured.Unstructured{Object: m}
}

func jobFrom(u *unstructured.Unstructured) batchv1.Job {
	var job batchv1.Job
	Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &job)).To(Succeed())
	return job
}

// jobWithStatus builds an unstructured Job carrying the given status, as Get would return.
func jobWithStatus(name, namespace string, status batchv1.JobStatus) *unstructured.Unstructured {
	return toUnstructured(&batchv1.Job{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Status:     status,
	})
}

var _ = Describe("JobRunner", func() {
	var (
		ctx    context.Context
		kube   *mocks.MockKubernetesClient
		runner client.JobRunner
		spec   client.JobSpec
	)

	BeforeEach(func() {
		ctx = context.Background()
		kube = new(mocks.MockKubernetesClient)
		runner = client.NewJobRunner(kube)
		spec = client.JobSpec{
			Name:    "s3-backup-x",
			Image:   "job-image:1",
			Command: []string{"/app/backup-job"},
			Labels:  map[string]string{"backup-id": "x"},
			Env:     map[string]string{"BACKUP_ID": "x"},
			Secrets: map[string]string{"STORE_SECRET_KEY": "sk"},
		}
	})

	AfterEach(func() { kube.AssertExpectations(GinkgoT()) })

	Describe("Run", func() {
		It("creates the secret, then the job with the right container, env and limits", func() {
			var createdJob *unstructured.Unstructured
			var createdSecret *corev1.Secret
			created := toUnstructured(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: spec.Name}})
			created.SetUID("job-uid-123")

			kube.On("CreateSecret", mock.Anything, "jobs-ns", mock.Anything).
				Run(func(a mock.Arguments) { createdSecret = a.Get(2).(*corev1.Secret) }).
				Return(&corev1.Secret{}, nil)
			kube.On("Create", mock.Anything, model.JobAPI, "jobs-ns", mock.Anything).
				Run(func(a mock.Arguments) { createdJob = a.Get(3).(*unstructured.Unstructured) }).
				Return(created, nil)
			kube.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			Expect(runner.Run(ctx, "jobs-ns", spec)).To(Succeed())

			job := jobFrom(createdJob)
			Expect(job.Namespace).To(Equal("jobs-ns"))
			Expect(*job.Spec.BackoffLimit).To(Equal(int32(3)))
			Expect(*job.Spec.TTLSecondsAfterFinished).To(Equal(int32(3600)))
			Expect(*job.Spec.ActiveDeadlineSeconds).To(Equal(int64(6 * 60 * 60)))

			c := job.Spec.Template.Spec.Containers[0]
			Expect(c.Image).To(Equal("job-image:1"))
			Expect(c.Command).To(Equal([]string{"/app/backup-job"}))
			Expect(job.Spec.Template.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyOnFailure))

			env := map[string]corev1.EnvVar{}
			for _, e := range c.Env {
				env[e.Name] = e
			}
			Expect(env["BACKUP_ID"].Value).To(Equal("x"))
			Expect(env["STORE_SECRET_KEY"].Value).To(BeEmpty())
			Expect(env["STORE_SECRET_KEY"].ValueFrom.SecretKeyRef.Name).To(Equal("s3-backup-x"))
			Expect(env["STORE_SECRET_KEY"].ValueFrom.SecretKeyRef.Key).To(Equal("STORE_SECRET_KEY"))

			Expect(createdSecret.Name).To(Equal("s3-backup-x"))
			Expect(createdSecret.StringData).To(HaveKeyWithValue("STORE_SECRET_KEY", "sk"))
		})

		It("adopts the secret under the created job for garbage collection", func() {
			var ref model.ResourceRef
			var patches []model.K8sPatch
			created := toUnstructured(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: spec.Name}})
			created.SetUID("job-uid-123")

			kube.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).Return(&corev1.Secret{}, nil)
			kube.On("Create", mock.Anything, model.JobAPI, mock.Anything, mock.Anything).Return(created, nil)
			kube.On("Patch", mock.Anything, mock.Anything, mock.Anything).
				Run(func(a mock.Arguments) {
					ref = a.Get(1).(model.ResourceRef)
					patches = a.Get(2).([]model.K8sPatch)
				}).Return(nil)

			Expect(runner.Run(ctx, "jobs-ns", spec)).To(Succeed())

			Expect(ref.APIResource).To(Equal(model.SecretAPI))
			Expect(ref.Name).To(Equal("s3-backup-x"))
			Expect(patches).To(HaveLen(1))
			owners := patches[0].Value.([]metav1.OwnerReference)
			Expect(owners[0].Kind).To(Equal("Job"))
			Expect(string(owners[0].UID)).To(Equal("job-uid-123"))
		})

		It("is a no-op if the job already exists", func() {
			kube.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).Return(&corev1.Secret{}, nil)
			kube.On("Create", mock.Anything, model.JobAPI, mock.Anything, mock.Anything).
				Return(nil, client.ErrResourceConflict)
			// No Patch: adoption is skipped when the job was not freshly created.

			Expect(runner.Run(ctx, "jobs-ns", spec)).To(Succeed())
			kube.AssertNotCalled(GinkgoT(), "Patch", mock.Anything, mock.Anything, mock.Anything)
		})

		It("creates no secret when the spec has none", func() {
			spec.Secrets = nil
			created := toUnstructured(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: spec.Name}})
			kube.On("Create", mock.Anything, model.JobAPI, mock.Anything, mock.Anything).Return(created, nil)

			Expect(runner.Run(ctx, "jobs-ns", spec)).To(Succeed())
			kube.AssertNotCalled(GinkgoT(), "CreateSecret", mock.Anything, mock.Anything, mock.Anything)
			kube.AssertNotCalled(GinkgoT(), "Patch", mock.Anything, mock.Anything, mock.Anything)
		})
	})

	Describe("State", func() {
		It("reports a succeeded job", func() {
			done := metav1.Now()
			kube.On("Get", mock.Anything, mock.Anything).Return(jobWithStatus("s3-backup-x", "jobs-ns", batchv1.JobStatus{
				CompletionTime: &done,
				Conditions:     []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}},
			}), nil)

			st, err := runner.State(ctx, "jobs-ns", "s3-backup-x")
			Expect(err).NotTo(HaveOccurred())
			Expect(st.Phase).To(Equal(client.JobSucceeded))
			Expect(st.FinishedAt).NotTo(BeNil())
		})

		It("reports a running job", func() {
			kube.On("Get", mock.Anything, mock.Anything).Return(jobWithStatus("s3-backup-x", "jobs-ns", batchv1.JobStatus{
				Active: 1,
			}), nil)

			st, err := runner.State(ctx, "jobs-ns", "s3-backup-x")
			Expect(err).NotTo(HaveOccurred())
			Expect(st.Phase).To(Equal(client.JobRunning))
		})

		It("reports a freshly created job as pending", func() {
			kube.On("Get", mock.Anything, mock.Anything).Return(jobWithStatus("s3-backup-x", "jobs-ns", batchv1.JobStatus{}), nil)

			st, err := runner.State(ctx, "jobs-ns", "s3-backup-x")
			Expect(err).NotTo(HaveOccurred())
			Expect(st.Phase).To(Equal(client.JobPending))
		})

		It("reports a failed job, enriching the reason with pod logs", func() {
			kube.On("Get", mock.Anything, mock.Anything).Return(jobWithStatus("s3-backup-x", "jobs-ns", batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Message: "BackoffLimitExceeded",
				}},
			}), nil)
			pod := toUnstructured(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "s3-backup-x-abc"}})
			kube.On("List", mock.Anything, mock.MatchedBy(func(o model.ListOptions) bool {
				return o.APIResource == model.PodAPI && o.LabelSelector == "job-name=s3-backup-x"
			})).Return(&unstructured.UnstructuredList{Items: []unstructured.Unstructured{*pod}}, nil)
			kube.On("GetPodLogs", mock.Anything, "jobs-ns", "s3-backup-x-abc", "job", int64(20)).
				Return("connection refused", nil)

			st, err := runner.State(ctx, "jobs-ns", "s3-backup-x")
			Expect(err).NotTo(HaveOccurred())
			Expect(st.Phase).To(Equal(client.JobFailed))
			Expect(st.Reason).To(Equal("BackoffLimitExceeded: connection refused"))
		})

		It("reports a missing job as not found", func() {
			kube.On("Get", mock.Anything, mock.Anything).Return(nil, client.ErrResourceNotFound)

			st, err := runner.State(ctx, "jobs-ns", "s3-backup-x")
			Expect(err).NotTo(HaveOccurred())
			Expect(st.Phase).To(Equal(client.JobNotFound))
		})
	})

	Describe("Delete", func() {
		It("deletes the job and its secret", func() {
			var deletedJob bool
			var deletedSecret string
			kube.On("Delete", mock.Anything, mock.MatchedBy(func(r model.ResourceRef) bool {
				return r.APIResource == model.JobAPI
			})).Run(func(mock.Arguments) { deletedJob = true }).Return(nil)
			kube.On("DeleteSecret", mock.Anything, "jobs-ns", mock.Anything).
				Run(func(a mock.Arguments) { deletedSecret = a.Get(2).(string) }).Return(nil)

			Expect(runner.Delete(ctx, "jobs-ns", "s3-backup-x")).To(Succeed())
			Expect(deletedJob).To(BeTrue())
			Expect(deletedSecret).To(Equal("s3-backup-x"))
		})

		It("tolerates a job and secret that do not exist", func() {
			kube.On("Delete", mock.Anything, mock.Anything).Return(client.ErrResourceNotFound)
			kube.On("DeleteSecret", mock.Anything, mock.Anything, mock.Anything).Return(client.ErrResourceNotFound)

			Expect(runner.Delete(ctx, "jobs-ns", "s3-backup-x")).To(Succeed())
		})
	})

	Describe("Replace", func() {
		It("deletes the existing job before dispatching a fresh one", func() {
			calls := []string{}
			kube.On("Delete", mock.Anything, mock.Anything).
				Run(func(mock.Arguments) { calls = append(calls, "delete-job") }).Return(nil)
			kube.On("DeleteSecret", mock.Anything, mock.Anything, mock.Anything).
				Run(func(mock.Arguments) { calls = append(calls, "delete-secret") }).Return(nil)
			kube.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
				Run(func(mock.Arguments) { calls = append(calls, "create-secret") }).Return(&corev1.Secret{}, nil)
			created := toUnstructured(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: spec.Name}})
			kube.On("Create", mock.Anything, model.JobAPI, mock.Anything, mock.Anything).
				Run(func(mock.Arguments) { calls = append(calls, "create-job") }).Return(created, nil)
			kube.On("Patch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			Expect(runner.Replace(ctx, "jobs-ns", spec)).To(Succeed())
			Expect(calls).To(Equal([]string{"delete-job", "delete-secret", "create-secret", "create-job"}))
		})
	})
})
