package model

// ManagedService is the base interface for all managed service types.
// It defines common operations that all managed service providers must support.
type ManagedService interface {
	// GetID returns the unique service identifier.
	GetID() ServiceID

	// GetConfig returns the service configuration.
	GetConfig() ServiceConfig

	// GetPlan returns the resource plan.
	GetPlan() Plan

	// GetSecrets returns the service secrets.
	GetSecrets() ServiceSecrets
}
