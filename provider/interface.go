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
//
// This follows the dependency inversion principle - the API layer depends
// on this abstraction rather than concrete implementations.
type Provider[CreateParams model.ManagedService, Status any, UpdateParams any] interface {
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

	// TakeBackup initiates a backup of the managed service.
	TakeBackup(ctx context.Context, args model.TakeBackupArgs) error

	// GetBackupStatus returns the status of a backup.
	GetBackupStatus(ctx context.Context, backupID string, retryArgs model.TakeBackupArgs) (BackupStatus, error)

	// DeleteBackup deletes a backup.
	DeleteBackup(ctx context.Context, args model.TakeBackupArgs) error
}

// BackupStatus represents the status of a backup operation.
type BackupStatus struct {
	// Phase is the current phase of the backup.
	Phase BackupPhase `json:"phase"`

	// StartedAt is when the backup started.
	StartedAt string `json:"startedAt,omitempty"`

	// CompletedAt is when the backup completed.
	CompletedAt string `json:"completedAt,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`
}

// BackupPhase represents the phase of a backup.
type BackupPhase string

// Backup phase constants.
const (
	BackupPhasePending   BackupPhase = "pending"
	BackupPhaseRunning   BackupPhase = "running"
	BackupPhaseCompleted BackupPhase = "completed"
	BackupPhaseFailed    BackupPhase = "failed"
)
