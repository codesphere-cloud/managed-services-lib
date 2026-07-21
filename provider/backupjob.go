// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"github.com/codesphere-cloud/managed-services-lib/client"
	"github.com/codesphere-cloud/managed-services-lib/model"
)

// BackupJobName is the name of the Job that takes a backup.
func BackupJobName(backupID model.BackupId) string {
	return ServiceJobName(JobOpBackup, string(backupID))
}

// DeleteBackupJobName is the name of the Job that deletes a backup.
func DeleteBackupJobName(backupID model.BackupId) string {
	return ServiceJobName(JobOpDeleteBackup, string(backupID))
}

// BackupStatusFromJob maps a Job snapshot to the backup status contract.
func BackupStatusFromJob(s client.JobState) BackupStatus {
	status := BackupStatus{Exists: s.Phase == client.JobSucceeded}
	if s.Phase == client.JobFailed {
		status.Error = s.Reason
	}
	return status
}
