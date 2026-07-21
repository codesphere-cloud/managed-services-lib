// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
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
		It("reports a succeeded job as existing, with no error", func() {
			s := provider.BackupStatusFromJob(client.JobState{Phase: client.JobSucceeded})
			Expect(s.Exists).To(BeTrue())
			Expect(s.Error).To(BeEmpty())
		})

		It("reports a failed job as not existing, carrying the reason into Error", func() {
			s := provider.BackupStatusFromJob(client.JobState{Phase: client.JobFailed, Reason: "boom"})
			Expect(s.Exists).To(BeFalse())
			Expect(s.Error).To(Equal("boom"))
		})

		It("reports running, pending and not-found jobs as not existing, no error", func() {
			for _, phase := range []client.JobPhase{client.JobRunning, client.JobPending, client.JobNotFound} {
				s := provider.BackupStatusFromJob(client.JobState{Phase: phase})
				Expect(s.Exists).To(BeFalse())
				Expect(s.Error).To(BeEmpty())
			}
		})
	})
})
