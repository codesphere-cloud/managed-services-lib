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
// Generic parameters:
//   - CreateParams: the full managed-service payload accepted on create
//   - Status: the per-service status payload returned to the marketplace
//   - UpdateParams: the partial update payload accepted on PATCH
type Provider[CreateParams any, Status any, UpdateParams any] interface {
	// Create creates a new managed service.
	Create(ctx context.Context, params CreateParams) error

	// List returns all service IDs managed by this provider.
	List(ctx context.Context) ([]model.ServiceID, error)

	// GetStatus returns the status of the specified services.
	// Services that don't exist are simply omitted from the result map.
	GetStatus(ctx context.Context, ids []model.ServiceID) (map[model.ServiceID]Status, error)

	// Update updates an existing managed service.
	Update(ctx context.Context, id model.ServiceID, args UpdateParams) error

	// Delete deletes a managed service.
	Delete(ctx context.Context, id model.ServiceID) error
}

// Backups is the optional backup capability, kept separate from Provider so a
// provider opts in by implementing it.
type Backups[BackupParams any] interface {
	// TakeBackup initiates a backup of the managed service.
	TakeBackup(ctx context.Context, backupID model.BackupId, params BackupParams) error

	// GetBackupStatus returns the status of a backup.
	GetBackupStatus(ctx context.Context, backupID model.BackupId, params BackupParams) (BackupStatus, error)

	// DeleteBackup deletes a backup.
	DeleteBackup(ctx context.Context, backupID model.BackupId, params BackupParams) error
}

// BackupStatus is the backup-status response contract expected by Codesphere:
// whether the backup exists (was taken successfully) and, if it failed, why.
type BackupStatus struct {
	// Exists is true once the backup has been taken successfully.
	Exists bool `json:"exists"`

	// Error contains the failure reason when the backup failed; empty otherwise.
	Error string `json:"error,omitempty"`
}
