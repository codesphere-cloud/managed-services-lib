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

// ServiceDetails contains common status details for a managed service.
type ServiceDetails struct {
	// Ready indicates if the service is ready to accept connections.
	Ready bool `json:"ready"`
}
