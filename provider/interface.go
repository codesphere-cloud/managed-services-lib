// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/codesphere-cloud/managed-services-lib/model"
)

// Provider defines the interface for managed service providers.
// Each provider (e.g., Postgres, FerretDB) implements this interface
// to handle its specific lifecycle operations.
//
// Whatever the REST contract defines is an explicit parameter; the type
// parameters are the provider's own schemas. Providers therefore never declare
// the id/teamId/customSubdomain fields or the plan.parameters wrapper themselves —
// the library decodes them off the request and passes them in.
//
// Generic parameters, each the contents of one provider-defined section of the
// contract:
//   - PlanParams: plan.parameters
//   - Config: config
//   - Secrets: secrets
//   - Details: details, the read-only part of the status response
//   - UpdateParams: the provider's partial PATCH payload
type Provider[PlanParams, Config, Secrets, Details, UpdateParams any] interface {
	// Create creates a new managed service.
	// recoverFrom is set when the request asks to restore the new service from a backup.
	Create(ctx context.Context, id model.ServiceID, teamID int, customSubdomain *string,
		plan PlanParams, config Config, secrets Secrets, recoverFrom *model.RecoverFrom) error

	// List returns all service IDs managed by this provider.
	List(ctx context.Context) ([]model.ServiceID, error)

	// GetStatus returns the status of the specified services.
	// Services that don't exist are simply omitted from the result map.
	GetStatus(ctx context.Context, ids []model.ServiceID) (map[model.ServiceID]model.ServiceStatus[PlanParams, Config, Details], error)

	// Update updates an existing managed service. args holds whichever of the
	// provider's own fields changed.
	Update(ctx context.Context, id model.ServiceID, teamID int, customSubdomain *string,
		args UpdateParams) error

	// Delete deletes a managed service.
	Delete(ctx context.Context, id model.ServiceID) error
}

// NewServiceStatus assembles a ServiceStatus, wrapping plan in the contract's
// plan.parameters envelope.
func NewServiceStatus[PlanParams, Config, Details any](
	plan PlanParams,
	config Config,
	details Details,
) model.ServiceStatus[PlanParams, Config, Details] {
	return model.ServiceStatus[PlanParams, Config, Details]{
		Plan:    model.PlanSpec[PlanParams]{Parameters: plan},
		Config:  config,
		Details: details,
	}
}

// Backups is the optional backup capability, kept separate from Provider so a
// provider opts in by implementing it. The type parameters are the provider's own
// backup-store schemas. retentionDays is nil when the request left retention
// unmanaged.
type Backups[BackupConfig, BackupSecrets any] interface {
	// TakeBackup initiates a backup of the managed service.
	TakeBackup(ctx context.Context, backupID model.BackupId, msID model.ServiceID, teamID int,
		config BackupConfig, secrets BackupSecrets, retentionDays *int) error

	// GetBackupStatus returns the status of a backup.
	GetBackupStatus(ctx context.Context, backupID model.BackupId, msID model.ServiceID, teamID int,
		config BackupConfig, secrets BackupSecrets, retentionDays *int) (model.BackupStatus, error)

	// DeleteBackup deletes a backup.
	DeleteBackup(ctx context.Context, backupID model.BackupId, msID model.ServiceID, teamID int,
		config BackupConfig, secrets BackupSecrets, retentionDays *int) error
}
