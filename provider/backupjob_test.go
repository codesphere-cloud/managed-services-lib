// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/codesphere-cloud/managed-services-lib/client"
	"github.com/codesphere-cloud/managed-services-lib/model"
	"github.com/codesphere-cloud/managed-services-lib/provider"
)

var _ = Describe("Backup job helpers", func() {
	Describe("Job names", func() {
		It("names the take- and delete-backup jobs by backup ID, distinctly", func() {
			Expect(provider.BackupJobName("bkp-7")).To(Equal("backup-bkp-7"))
			Expect(provider.DeleteBackupJobName("bkp-7")).To(Equal("delete-backup-bkp-7"))
			Expect(provider.BackupJobName("bkp-7")).NotTo(Equal(provider.DeleteBackupJobName("bkp-7")))
		})

		It("agrees with the name a ServiceJob of that operation produces", func() {
			spec := provider.ServiceJobSpec(provider.ServiceJob{
				Operation: provider.JobOpBackup, MsID: "svc-42", Key: "bkp-7",
			})
			Expect(spec.Name).To(Equal(provider.BackupJobName("bkp-7")))
			// The backup operation auto-stamps the backup identity label.
			Expect(spec.Labels).To(HaveKeyWithValue(model.ServiceIDLabel, "svc-42"))
			Expect(spec.Labels).To(HaveKeyWithValue(model.BackupIDLabel, "bkp-7"))
		})
	})

	Describe("BackupStatusFromJob", func() {
		started := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
		finished := time.Date(2026, 7, 17, 10, 5, 0, 0, time.UTC)

		It("maps a running job", func() {
			s := provider.BackupStatusFromJob(client.JobState{Phase: client.JobRunning, StartedAt: &started})
			Expect(s.Phase).To(Equal(provider.BackupPhaseRunning))
			Expect(s.StartedAt).To(Equal("2026-07-17T10:00:00Z"))
			Expect(s.Error).To(BeEmpty())
		})

		It("maps a succeeded job with timestamps", func() {
			s := provider.BackupStatusFromJob(client.JobState{
				Phase: client.JobSucceeded, StartedAt: &started, FinishedAt: &finished,
			})
			Expect(s.Phase).To(Equal(provider.BackupPhaseCompleted))
			Expect(s.CompletedAt).To(Equal("2026-07-17T10:05:00Z"))
		})

		It("maps a failed job, carrying the reason into Error", func() {
			s := provider.BackupStatusFromJob(client.JobState{Phase: client.JobFailed, Reason: "boom"})
			Expect(s.Phase).To(Equal(provider.BackupPhaseFailed))
			Expect(s.Error).To(Equal("boom"))
		})

		It("maps pending and not-found to pending", func() {
			Expect(provider.BackupStatusFromJob(client.JobState{Phase: client.JobPending}).Phase).
				To(Equal(provider.BackupPhasePending))
			Expect(provider.BackupStatusFromJob(client.JobState{Phase: client.JobNotFound}).Phase).
				To(Equal(provider.BackupPhasePending))
		})
	})
})
