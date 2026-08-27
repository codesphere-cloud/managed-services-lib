// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"encoding/json"
)

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
}

// RecoverFrom, sent only on create, asks the provider to restore the new
// service from an existing backup instead of provisioning it empty. Config
// and Secrets are deferred as raw JSON since they are provider specific.
type RecoverFrom struct {
	ID      BackupId        `json:"id"`
	Config  json.RawMessage `json:"config"`
	Secrets json.RawMessage `json:"secrets"`
}

// BackupStatus is the backup-status response contract expected by Codesphere:
// whether the backup exists (was taken successfully) and, if it failed, why.
type BackupStatus struct {
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}
