// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/codesphere-cloud/managed-services-lib/model"
)

// CreateRequest is a decoded create call: the service-level fields the contract
// defines, plus the provider's own sections. Providers never declare the
// id/teamId/customSubdomain fields or the plan.parameters wrapper themselves —
// the library decodes them off the request and fills them in here.
type CreateRequest[PlanParams, Config, Secrets any] struct {
	ID              model.ServiceID
	TeamID          int
	CustomSubdomain *string
	Plan            PlanParams
	Config          Config
	Secrets         Secrets
	RecoverFrom     *model.RecoverFrom
}

// UpdateRequest is a decoded PATCH: the service-level fields the contract
// defines, plus the provider's own partial payload in Params.
type UpdateRequest[UpdateParams any] struct {
	ID              model.ServiceID
	TeamID          int
	CustomSubdomain *string
	Pause           *bool
	Params          UpdateParams
}

// BackupRequest is a decoded backup call. The same shape serves taking, polling
// and deleting a backup, since Codesphere sends the backup store's coordinates
// with each one.
type BackupRequest[BackupConfig, BackupSecrets any] struct {
	BackupID      model.BackupId
	ServiceID     model.ServiceID
	TeamID        int
	Config        BackupConfig
	Secrets       BackupSecrets
	RetentionDays *int
}

// Provider defines the interface for managed service providers.
// Each provider (e.g., Postgres, FerretDB) implements this interface
// to handle its specific lifecycle operations.
//
// Whatever the REST contract defines arrives in a request struct; the type
// parameters are the provider's own schemas.
//
// Generic parameters, each the contents of one provider-defined section of the
// contract:
//   - PlanParams: plan.parameters
//   - Config: config
//   - Secrets: secrets
//   - Details: details, the read-only part of the status response
//   - UpdateParams: the provider's partial PATCH payload
type Provider[PlanParams, Config, Secrets, Details, UpdateParams any] interface {
	// Create creates a new managed service. A new service always comes up
	// running, so CreateRequest carries no pause flag; pausing is a later Update.
	Create(ctx context.Context, req CreateRequest[PlanParams, Config, Secrets]) error

	// List returns all service IDs managed by this provider.
	List(ctx context.Context) ([]model.ServiceID, error)

	// GetStatus returns the status of the specified services.
	// Services that don't exist are simply omitted from the result map.
	GetStatus(ctx context.Context, ids []model.ServiceID) (map[model.ServiceID]model.ServiceStatus[PlanParams, Config, Details], error)

	// Update updates an existing managed service.
	Update(ctx context.Context, req UpdateRequest[UpdateParams]) error

	// Delete deletes a managed service.
	Delete(ctx context.Context, id model.ServiceID) error
}

// NewServiceStatus assembles a ServiceStatus, wrapping plan in the contract's
// plan.parameters envelope.
func NewServiceStatus[PlanParams, Config, Details any](
	plan PlanParams,
	config Config,
	details Details,
	pause *bool,
	err *string,

) model.ServiceStatus[PlanParams, Config, Details] {
	return model.ServiceStatus[PlanParams, Config, Details]{
		Plan:    model.PlanSpec[PlanParams]{Parameters: plan},
		Config:  config,
		Details: details,
		Pause:   pause,
		Error:   err,
	}
}

// Backups is the optional backup capability, kept separate from Provider so a
// provider opts in by implementing it. The type parameters are the provider's own
// backup-store schemas.
type Backups[BackupConfig, BackupSecrets any] interface {
	// TakeBackup initiates a backup of the managed service.
	TakeBackup(ctx context.Context, req BackupRequest[BackupConfig, BackupSecrets]) error

	// GetBackupStatus returns the status of a backup.
	GetBackupStatus(ctx context.Context, req BackupRequest[BackupConfig, BackupSecrets]) (model.BackupStatus, error)

	// DeleteBackup deletes a backup.
	DeleteBackup(ctx context.Context, req BackupRequest[BackupConfig, BackupSecrets]) error
}
