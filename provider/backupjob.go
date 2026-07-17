// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"github.com/codesphere-cloud/managed-services-lib/client"
)

// BackupJobName is the name of the Job that takes a backup.
func BackupJobName(backupID string) string {
	return ServiceJobName(JobOpBackup, backupID)
}

// DeleteBackupJobName is the name of the Job that deletes a backup.
func DeleteBackupJobName(backupID string) string {
	return ServiceJobName(JobOpDeleteBackup, backupID)
}

// BackupStatusFromJob maps a Job snapshot onto the backup status contract. A Job
// that no longer exists is reported as pending — a provider that needs to
// distinguish "never taken" from "completed and garbage-collected" should check
// the JobState phase directly before calling this.
func BackupStatusFromJob(s client.JobState) BackupStatus {
	op := OperationStatusFromJob(s)
	return BackupStatus{
		Phase:       BackupPhase(op.Phase),
		StartedAt:   op.StartedAt,
		CompletedAt: op.CompletedAt,
		Error:       op.Error,
	}
}
