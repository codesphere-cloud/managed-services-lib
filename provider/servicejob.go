// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"maps"
	"time"

	"github.com/codesphere-cloud/managed-services-lib/client"
	"github.com/codesphere-cloud/managed-services-lib/model"
)

// Operation prefixes for detached service Jobs. Each becomes the Job name prefix
// and identifies the kind of work. Providers running other detached work can use
// their own prefix with ServiceJob directly.
const (
	JobOpBackup       = "backup"
	JobOpDeleteBackup = "delete-backup"
	JobOpRestore      = "restore"
)

// operationKeyLabels maps an operation to the identity label its Key is stamped
// under, so every Job of that kind carries a consistent, discoverable label
// without the caller having to remember it. Operations absent from this map
// (restore, provider-defined kinds) carry only the ServiceIDLabel.
var operationKeyLabels = map[string]string{
	JobOpBackup:       model.BackupIDLabel,
	JobOpDeleteBackup: model.BackupIDLabel,
}

// ServiceJob describes a detached one-shot operation run on behalf of a managed
// service — a backup, backup deletion, restore-on-create and so on.
// It is the single entry point for dispatching such work: it applies the Job
// conventions (naming and identity labels) so every provider's operations are
// named and labelled consistently. Execution details (Image/Command/Env/Secrets)
// pass through verbatim: each provider's operation image defines its own contract.
type ServiceJob struct {
	// Operation is the job kind; it becomes the Job name prefix (one of the JobOp
	// constants, or a provider-defined prefix). Known operations also fix which
	// identity label the Key is stamped under (see operationKeyLabels).
	Operation string
	// MsID is the service the job acts on. It is always stamped as ServiceIDLabel
	// so the Job is discoverable and cleaned up with the service.
	MsID string
	// Key uniquely identifies the job within its Operation, becoming the Job name
	// suffix and idempotency key. Use the backup ID for per-backup operations, or
	// the MsID for whole-service operations (e.g. restore).
	Key string
	// Image is the container image the Job runs.
	Image string
	// Command overrides the image entrypoint.
	Command []string
	// Env holds plain (non-secret) environment variables.
	Env map[string]string
	// Secrets holds secret environment variables. Each is stored in the Job's
	// owned Secret and injected via a secretKeyRef, so values never appear in the
	// Job or Pod manifest.
	Secrets map[string]string
	// Labels are merged under the identity labels, which the caller cannot
	// override.
	Labels map[string]string
}

// ServiceJobName is the Job naming convention: "<operation>-<key>". It is stable
// per operation and key so Run is idempotent and State/Delete resolve the same
// Job. Keep keys short: the result must stay within the 63-char DNS-1123 limit.
func ServiceJobName(operation, key string) string {
	return operation + "-" + key
}

// ServiceJobSpec applies the naming and identity-label conventions to build a
// client.JobSpec. It always stamps ServiceIDLabel from MsID, and — for
// operations with a registered key label (see operationKeyLabels) — the Key
// under that label. Caller Labels ride along underneath and cannot override the
// identity labels.
func ServiceJobSpec(j ServiceJob) client.JobSpec {
	labels := map[string]string{}
	maps.Copy(labels, j.Labels)
	// Identity labels are authoritative — set them last so caller labels cannot
	// override them.
	labels[model.ServiceIDLabel] = j.MsID
	if keyLabel := operationKeyLabels[j.Operation]; keyLabel != "" {
		labels[keyLabel] = j.Key
	}
	return client.JobSpec{
		Name:    ServiceJobName(j.Operation, j.Key),
		Image:   j.Image,
		Command: j.Command,
		Env:     j.Env,
		Secrets: j.Secrets,
		Labels:  labels,
	}
}

// OperationStatus is the phase snapshot shared by every detached operation. It
// is derived from a client.JobState. The backup REST contract has its own
// BackupStatus (built from this); other operations can surface it directly.
type OperationStatus struct {
	// Phase is the current phase of the operation.
	Phase OperationPhase `json:"phase"`

	// StartedAt is when the operation started.
	StartedAt string `json:"startedAt,omitempty"`

	// CompletedAt is when the operation completed.
	CompletedAt string `json:"completedAt,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`
}

// OperationPhase represents the phase of a detached operation. Its values match
// the backup REST contract's phases so BackupStatus can be built from it.
type OperationPhase string

// Operation phase constants.
const (
	OperationPhasePending   OperationPhase = "pending"
	OperationPhaseRunning   OperationPhase = "running"
	OperationPhaseCompleted OperationPhase = "completed"
	OperationPhaseFailed    OperationPhase = "failed"
)

// OperationStatusFromJob maps a Job snapshot onto an OperationStatus. A Job that
// no longer exists is reported as pending — a caller that needs to distinguish
// "never run" from "completed and garbage-collected" should check the JobState
// phase directly before calling this.
func OperationStatusFromJob(s client.JobState) OperationStatus {
	status := OperationStatus{
		Phase:       OperationPhasePending,
		StartedAt:   formatTime(s.StartedAt),
		CompletedAt: formatTime(s.FinishedAt),
	}
	switch s.Phase {
	case client.JobRunning:
		status.Phase = OperationPhaseRunning
	case client.JobSucceeded:
		status.Phase = OperationPhaseCompleted
	case client.JobFailed:
		status.Phase = OperationPhaseFailed
		status.Error = s.Reason
	}
	return status
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
