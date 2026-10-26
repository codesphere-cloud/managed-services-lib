// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// ServiceID is a unique identifier for a managed service instance.
type ServiceID string

// BackupId is a unique identifier for a managed service backup.
type BackupId string

// PlanSpec is the {"parameters": ...} envelope the contract wraps a plan in.
type PlanSpec[Params any] struct {
	Parameters Params `json:"parameters"`
}

// ServiceStatus is the per-service value of the status response. Build it with
// NewServiceStatus, which applies the contract's plan.parameters wrapper.
type ServiceStatus[PlanParams, Config, Details any] struct {
	Plan    PlanSpec[PlanParams] `json:"plan"`
	Config  Config               `json:"config"`
	Details Details              `json:"details"`
	Pause   *bool                `json:"pause,omitempty"`
	Error   *string              `json:"error,omitempty"`
	Backups *ServiceBackups      `json:"backups,omitempty"`
}

// RecoverFrom, sent only on create, asks the provider to restore the new
// service instead of provisioning it empty: either from the backup ID, or
// (point-in-time recovery) from service MsID's backups as close to Time as
// possible. MsID does not need to exist anymore.
type RecoverFrom struct {
	ID      BackupId           `json:"id,omitempty" binding:"required_without=MsID,excluded_with=MsID"`
	MsID    ServiceID          `json:"msId,omitempty" binding:"required_without=ID"`
	Time    *time.Time         `json:"time,omitempty" binding:"required_with=MsID,excluded_without=MsID"`
	Config  BackupStoreConfig  `json:"config"`
	Secrets BackupStoreSecrets `json:"secrets"`
}

// ServiceBackups is the contract's service-level backups section, shared by the
// update payload and the status response.
type ServiceBackups struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// BackupStatus is the backup-status response contract expected by Codesphere:
// whether the backup exists (was taken successfully) and, if it failed, why.
type BackupStatus struct {
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

// BackupStoreConfig is the backup store config Codesphere sends on backup requests
// and in recoverFrom.
type BackupStoreConfig struct {
	Type        string `json:"type" binding:"required,eq=s3"`
	EndpointURL string `json:"endpointUrl" binding:"required,url"`
	// Permissive: up to 255 characters is the union of the S3-compatible
	// naming rules (Ceph's relaxed names include AWS and GCS names).
	// Store-specific rules are left to the store.
	Bucket      string `json:"bucket" binding:"required,max=255"`
	Prefix      string `json:"prefix,omitempty" binding:"startsnotwith=/,endsnotwith=/"`
	AccessKeyID string `json:"accessKeyId" binding:"required"`
}

// BackupStoreSecrets holds the backup store's secret key.
type BackupStoreSecrets struct {
	SecretKey string `json:"secretKey" binding:"required"`
}

// Path returns the store location as bucket[/prefix].
func (s BackupStoreConfig) Path() string {
	if s.Prefix == "" {
		return s.Bucket
	}
	return s.Bucket + "/" + s.Prefix
}
