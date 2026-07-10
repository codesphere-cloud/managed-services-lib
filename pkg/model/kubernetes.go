package model

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIResource identifies a Kubernetes API resource.
type APIResource struct {
	// Group is the API group.
	Group string

	// Version is the API version.
	Version string

	// Plural is the plural resource name.
	Plural string
}

// ResourceRef identifies a specific Kubernetes resource.
type ResourceRef struct {
	APIResource

	// Name is the resource name.
	Name string

	// Namespace is the resource namespace (empty for cluster-scoped resources).
	Namespace string
}

// ListOptions specifies options for listing resources.
type ListOptions struct {
	APIResource

	// Namespace limits the list to a specific namespace.
	Namespace string

	// LabelSelector filters by label selector.
	LabelSelector string

	// FieldSelector filters by field selector.
	FieldSelector string
}

// K8sPatch represents a JSON patch operation.
type K8sPatch struct {
	// Op is the operation (add, remove, replace, move, copy, test).
	Op string `json:"op"`

	// Path is the JSON pointer to the target location.
	Path string `json:"path"`

	// Value is the value for add/replace operations.
	Value interface{} `json:"value,omitempty"`
}

// OwnerReference wraps Kubernetes owner reference.
type OwnerReference = metav1.OwnerReference

// Common Kubernetes API resources.
var (
	SecretAPI = APIResource{
		Group:   "",
		Version: "v1",
		Plural:  "secrets",
	}

	PodAPI = APIResource{
		Group:   "",
		Version: "v1",
		Plural:  "pods",
	}

	ServiceAPI = APIResource{
		Group:   "",
		Version: "v1",
		Plural:  "services",
	}

	DeploymentAPI = APIResource{
		Group:   "apps",
		Version: "v1",
		Plural:  "deployments",
	}

	StatefulSetAPI = APIResource{
		Group:   "apps",
		Version: "v1",
		Plural:  "statefulsets",
	}

	PersistentVolumeClaimAPI = APIResource{
		Group:   "",
		Version: "v1",
		Plural:  "persistentvolumeclaims",
	}

	JobAPI = APIResource{
		Group:   "batch",
		Version: "v1",
		Plural:  "jobs",
	}
)

// Codesphere label keys for managed service resources.
const (
	// LabelPrefix is the domain prefix for all managed service labels and annotations.
	LabelPrefix = "managed-services.codesphere.com"

	// ServiceIDLabel is the label key for the managed service ID.
	ServiceIDLabel = LabelPrefix + "/id"

	// ProviderLabel is the label key for the managed service provider type.
	ProviderLabel = LabelPrefix + "/provider"

	// TeamIDLabel is the label key for the Codesphere team ID.
	TeamIDLabel = LabelPrefix + "/team-id"
)
