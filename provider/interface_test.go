// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/codesphere-cloud/managed-services-lib/model"
	"github.com/codesphere-cloud/managed-services-lib/provider"
)

type MockPlan struct{ Replicas int }
type MockPlanUpdate struct{ Replicas *int }
type MockConfig struct{ Version string }
type MockConfigUpdate struct{ Version *string }
type MockSecrets struct{ Password string }
type MockSecretsUpdate struct{ Password *string }
type MockDetails struct{ Ready bool }

type MockUpdateArgs struct {
	Plan    *MockPlanUpdate
	Config  *MockConfigUpdate
	Secrets *MockSecretsUpdate
}

type MockBackupConfig struct{ Bucket string }
type MockBackupSecrets struct{ Token string }

type MockProvider struct{}

// Compile-time checks for interface compatibility
var _ provider.Provider[MockPlan, MockConfig, MockSecrets, MockDetails, MockUpdateArgs] = (*MockProvider)(nil)
var _ provider.Backups[MockBackupConfig, MockBackupSecrets] = (*MockProvider)(nil)

type CreateRequest = provider.CreateRequest[MockPlan, MockConfig, MockSecrets]
type UpdateRequest = provider.UpdateRequest[MockUpdateArgs]
type Status = model.ServiceStatus[MockPlan, MockConfig, MockDetails]
type BackupRequest = provider.BackupRequest[MockBackupConfig, MockBackupSecrets]

func (m *MockProvider) Create(ctx context.Context, req CreateRequest) error {
	return nil
}

func (m *MockProvider) List(ctx context.Context) ([]model.ServiceID, error) {
	return []model.ServiceID{"svc-test-1"}, nil
}

func (m *MockProvider) GetStatus(ctx context.Context, ids []model.ServiceID) (map[model.ServiceID]Status, error) {
	return make(map[model.ServiceID]model.ServiceStatus[MockPlan, MockConfig, MockDetails]), nil
}

func (m *MockProvider) Update(ctx context.Context, req UpdateRequest) error {
	return nil
}

func (m *MockProvider) Delete(ctx context.Context, id model.ServiceID) error {
	return nil
}

func (m *MockProvider) TakeBackup(ctx context.Context, req BackupRequest) error {
	return nil
}

func (m *MockProvider) GetBackupStatus(ctx context.Context, req BackupRequest) (model.BackupStatus, error) {
	return model.BackupStatus{}, nil
}

func (m *MockProvider) DeleteBackup(ctx context.Context, req BackupRequest) error {
	return nil
}

var _ = Describe("Provider API Contract", func() {
	var (
		ctx context.Context
		p   *MockProvider
	)

	BeforeEach(func() {
		ctx = context.Background()
		p = &MockProvider{}
	})

	It("compiles and accepts the expected CreateRequest shape", func() {
		req := provider.CreateRequest[MockPlan, MockConfig, MockSecrets]{
			ID:              "svc-123",
			TeamID:          456,
			CustomSubdomain: new("custom.example.com"),
			Plan:            MockPlan{Replicas: 3},
			Config:          MockConfig{Version: "3.7"},
			Secrets:         MockSecrets{Password: "secret"},
			RecoverFrom:     nil,
		}

		err := p.Create(ctx, req)
		Expect(err).ToNot(HaveOccurred())
	})

	It("compiles and accepts the expected UpdateRequest shape with UpdateArgs params", func() {
		req := provider.UpdateRequest[MockUpdateArgs]{
			ID:              "svc-123",
			TeamID:          456,
			CustomSubdomain: nil,       // Omitted
			Pause:           new(true), // Provided
			Params: MockUpdateArgs{
				Plan:    &MockPlanUpdate{Replicas: new(1)}, // Provided via pointer
				Config:  nil,                               // Omitted
				Secrets: nil,                               // Omitted
			},
		}

		err := p.Update(ctx, req)
		Expect(err).ToNot(HaveOccurred())
	})

	It("compiles and accepts the expected BackupRequest shape", func() {
		req := provider.BackupRequest[MockBackupConfig, MockBackupSecrets]{
			BackupID:      "backup-789",
			ServiceID:     "svc-123",
			TeamID:        456,
			Config:        MockBackupConfig{Bucket: "s3-bucket"},
			Secrets:       MockBackupSecrets{Token: "xyz"},
			RetentionDays: new(30),
		}

		err := p.TakeBackup(ctx, req)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewServiceStatus", func() {
		It("safely populates both pointer fields when provided", func() {
			status := provider.NewServiceStatus(
				MockPlan{Replicas: 3},
				MockConfig{Version: "3.7"},
				MockDetails{Ready: true},
				new(false),
				nil,
			)

			Expect(status.Plan.Parameters.Replicas).To(Equal(3))
			Expect(status.Config.Version).To(Equal("3.7"))
			Expect(status.Details.Ready).To(BeTrue())

			Expect(*status.Pause).To(BeFalse())
			Expect(status.Error).To(BeNil())
		})
	})
})
