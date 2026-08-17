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
// Whatever Codesphere sends or structures is an explicit parameter; the type
// parameters are the provider's own schemas. Providers therefore never declare
// the id/teamId/customSubdomain fields or the plan.parameters wrapper themselves.
//
// Generic parameters, each the contents of one provider-defined section of the
// REST contract:
//   - PlanParams: plan.parameters
//   - Config: config
//   - Secrets: secrets
//   - Details: details, the read-only part of the status response
//   - UpdateParams: the provider's partial PATCH payload
//
// Providers are expected to alias the instantiation once:
//
//	type MyProvider = provider.Provider[Params, Config, Secrets, Details, UpdateParams]
type Provider[PlanParams, Config, Secrets, Details, UpdateParams any] interface {
	// Create creates a new managed service.
	Create(ctx context.Context, id model.ServiceID, teamID int, customSubdomain *string,
		plan PlanParams, config Config, secrets Secrets) error

	// List returns all service IDs managed by this provider.
	List(ctx context.Context) ([]model.ServiceID, error)

	// GetStatus returns the status of the specified services.
	// Services that don't exist are simply omitted from the result map.
	GetStatus(ctx context.Context, ids []model.ServiceID) (map[model.ServiceID]ServiceStatus[PlanParams, Config, Details], error)

	// Update updates an existing managed service. args holds whichever of the
	// provider's own fields changed.
	Update(ctx context.Context, id model.ServiceID, teamID int, customSubdomain *string,
		args UpdateParams) error

	// Delete deletes a managed service.
	Delete(ctx context.Context, id model.ServiceID) error
}

// ServiceStatus is the per-service value of the status response. Build it with
// NewServiceStatus, which applies the contract's plan.parameters wrapper.
type ServiceStatus[PlanParams, Config, Details any] struct {
	// Plan echoes the service's current plan parameters.
	Plan planSpec[PlanParams] `json:"plan"`

	// Config echoes the service's current configuration.
	Config Config `json:"config"`

	// Details is read-only provider data (hostnames, ports, readiness, ...).
	Details Details `json:"details"`
}

// NewServiceStatus assembles a ServiceStatus, wrapping plan in the contract's
// plan.parameters envelope.
func NewServiceStatus[PlanParams, Config, Details any](
	plan PlanParams,
	config Config,
	details Details,
) ServiceStatus[PlanParams, Config, Details] {
	return ServiceStatus[PlanParams, Config, Details]{
		Plan:    planSpec[PlanParams]{Parameters: plan},
		Config:  config,
		Details: details,
	}
}

// Backups is the optional backup capability, kept separate from Provider so a
// provider opts in by implementing it. The type parameters are the provider's own
// backup-store schemas.
type Backups[BackupConfig, BackupSecrets any] interface {
	// TakeBackup initiates a backup of the managed service.
	TakeBackup(ctx context.Context, backupID model.BackupId, msID model.ServiceID,
		config BackupConfig, secrets BackupSecrets) error

	// GetBackupStatus returns the status of a backup.
	GetBackupStatus(ctx context.Context, backupID model.BackupId, msID model.ServiceID,
		config BackupConfig, secrets BackupSecrets) (BackupStatus, error)

	// DeleteBackup deletes a backup.
	DeleteBackup(ctx context.Context, backupID model.BackupId, msID model.ServiceID,
		config BackupConfig, secrets BackupSecrets) error
}

// BackupStatus is the backup-status response contract expected by Codesphere:
// whether the backup exists (was taken successfully) and, if it failed, why.
type BackupStatus struct {
	// Exists is true once the backup has been taken successfully.
	Exists bool `json:"exists"`

	// Error contains the failure reason when the backup failed; empty otherwise.
	Error string `json:"error,omitempty"`
}
