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

var _ = Describe("Service job core", func() {
	Describe("ServiceJobSpec", func() {
		It("names the job <operation>-<key> and always stamps the service label", func() {
			spec := provider.ServiceJobSpec(provider.ServiceJob{
				Operation: "something",
				MsID:      "svc-42",
				Key:       "svc-42",
				Image:     "something:1",
			})

			Expect(spec.Name).To(Equal("something-svc-42"))
			Expect(spec.Name).To(Equal(provider.ServiceJobName("something", "svc-42")))
			Expect(spec.Labels).To(HaveKeyWithValue(model.ServiceIDLabel, "svc-42"))
		})

		It("auto-stamps the key label for operations that have one", func() {
			// Backup-family operations stamp the backup identity label from Key...
			for _, op := range []string{provider.JobOpBackup, provider.JobOpDeleteBackup} {
				spec := provider.ServiceJobSpec(provider.ServiceJob{Operation: op, MsID: "svc-42", Key: "bkp-7"})
				Expect(spec.Labels).To(HaveKeyWithValue(model.BackupIDLabel, "bkp-7"))
			}

			// ...operations without a registered label (restore, custom) do not.
			for _, op := range []string{provider.JobOpRestore, "custom"} {
				spec := provider.ServiceJobSpec(provider.ServiceJob{Operation: op, MsID: "svc-42", Key: "svc-42"})
				Expect(spec.Labels).NotTo(HaveKey(model.BackupIDLabel))
			}
		})

		It("lets caller labels ride along but never override identity labels", func() {
			spec := provider.ServiceJobSpec(provider.ServiceJob{
				Operation: provider.JobOpBackup, MsID: "svc-42", Key: "bkp-7",
				Labels: map[string]string{
					"custom":             "yes",
					model.ServiceIDLabel: "hijack",
					model.BackupIDLabel:  "hijack",
				},
			})

			Expect(spec.Labels).To(HaveKeyWithValue("custom", "yes"))
			Expect(spec.Labels).To(HaveKeyWithValue(model.ServiceIDLabel, "svc-42"))
			Expect(spec.Labels).To(HaveKeyWithValue(model.BackupIDLabel, "bkp-7"))
		})
	})

	Describe("OperationStatusFromJob", func() {
		started := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
		finished := time.Date(2026, 7, 17, 10, 5, 0, 0, time.UTC)

		It("maps a running job", func() {
			s := provider.OperationStatusFromJob(client.JobState{Phase: client.JobRunning, StartedAt: &started})
			Expect(s.Phase).To(Equal(provider.OperationPhaseRunning))
			Expect(s.StartedAt).To(Equal("2026-07-17T10:00:00Z"))
			Expect(s.Error).To(BeEmpty())
		})

		It("maps a succeeded job with timestamps", func() {
			s := provider.OperationStatusFromJob(client.JobState{
				Phase: client.JobSucceeded, StartedAt: &started, FinishedAt: &finished,
			})
			Expect(s.Phase).To(Equal(provider.OperationPhaseCompleted))
			Expect(s.CompletedAt).To(Equal("2026-07-17T10:05:00Z"))
		})

		It("maps a failed job, carrying the reason into Error", func() {
			s := provider.OperationStatusFromJob(client.JobState{Phase: client.JobFailed, Reason: "boom"})
			Expect(s.Phase).To(Equal(provider.OperationPhaseFailed))
			Expect(s.Error).To(Equal("boom"))
		})

		It("maps pending and not-found to pending", func() {
			Expect(provider.OperationStatusFromJob(client.JobState{Phase: client.JobPending}).Phase).
				To(Equal(provider.OperationPhasePending))
			Expect(provider.OperationStatusFromJob(client.JobState{Phase: client.JobNotFound}).Phase).
				To(Equal(provider.OperationPhasePending))
		})
	})
})
