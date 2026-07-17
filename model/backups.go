// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package model

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

// TakeBackupArgs contains arguments for taking a backup.
type TakeBackupArgs struct {
	// ID is the backup ID.
	ID string `json:"id"`

	// MsID is the managed service ID.
	MsID string `json:"msId"`

	// Config is the backup store configuration.
	Config BackupStoreConfig `json:"config"`

	// Secrets is the backup store secrets.
	Secrets BackupStoreSecrets `json:"secrets"`
}
