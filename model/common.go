// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package model

// ServiceID is a unique identifier for a managed service instance.
type ServiceID string

// PlanParameters defines the resource allocation for a managed service.
type PlanParameters struct {
	// StorageMiB is the storage size in MiB.
	StorageMiB int `json:"storage"`

	// CPUTenths is the CPU allocation in tenths of a core.
	CPUTenths int `json:"cpu"`

	// MemoryMiB is the memory allocation in MiB.
	MemoryMiB int `json:"memory"`
}

// Plan wraps the plan parameters.
type Plan struct {
	Parameters PlanParameters `json:"parameters"`
}

// ServiceConfig holds configuration for a managed service.
type ServiceConfig struct {
	// Version is the version of the managed service.
	Version string `json:"version"`
}

// ServiceSecrets holds sensitive data for a managed service.
type ServiceSecrets struct {
	// SuperuserPassword is the superuser/admin password.
	SuperuserPassword string `json:"superuserPassword"`
}

// BackupStoreConfig holds configuration for backup storage.
type BackupStoreConfig struct {
	// EndpointURL is the S3-compatible endpoint URL.
	EndpointURL string `json:"endpointUrl"`

	// DestinationPath is the path/bucket for backups.
	DestinationPath string `json:"destinationPath"`

	// AccessKeyID is the access key ID for the backup store.
	AccessKeyID string `json:"accessKeyId"`
}

// BackupStoreSecrets holds sensitive data for backup storage.
type BackupStoreSecrets struct {
	// SecretKey is the secret access key.
	SecretKey string `json:"secretKey"`
}

// RecoverFromBackup specifies recovery from a specific backup.
type RecoverFromBackup struct {
	// ID is the backup ID to recover from.
	ID string `json:"id"`

	// Config is the backup store configuration.
	Config BackupStoreConfig `json:"config"`

	// Secrets is the backup store secrets.
	Secrets BackupStoreSecrets `json:"secrets"`
}

// RecoverFromTimestamp specifies point-in-time recovery.
type RecoverFromTimestamp struct {
	// MsID is the managed service ID to recover from.
	MsID string `json:"msId"`

	// Time is the target timestamp in ISO 8601 format.
	Time string `json:"time"`

	// Config is the backup store configuration.
	Config BackupStoreConfig `json:"config"`

	// Secrets is the backup store secrets.
	Secrets BackupStoreSecrets `json:"secrets"`
}

// ServiceDetails contains common status details for a managed service.
type ServiceDetails struct {
	// Ready indicates if the service is ready to accept connections.
	Ready bool `json:"ready"`
}

// TakeBackupArgs contains arguments for taking a backup.
type TakeBackupArgs struct {
	// ID is the service ID.
	ID string `json:"id"`

	// Config is the backup store configuration.
	Config BackupStoreConfig `json:"config"`

	// Secrets is the backup store secrets.
	Secrets BackupStoreSecrets `json:"secrets"`
}
