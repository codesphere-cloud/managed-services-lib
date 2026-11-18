// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"regexp"
	"strings"

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

	// NameLabel is the label key for the managed service name.
	NameLabel = LabelPrefix + "/name"

	// BackupIDLabel is the label key for a backup ID. Applied to backup Jobs
	// (alongside ServiceIDLabel) so a service's backups are discoverable.
	BackupIDLabel = LabelPrefix + "/backup-id"

	// ProviderLabel is the label key for the managed service provider type.
	ProviderLabel = LabelPrefix + "/provider"

	// TeamIDLabel is the label key for the Codesphere team ID.
	TeamIDLabel = LabelPrefix + "/team-id"
)

// maxLabelValueLength is the Kubernetes limit for label values.
const maxLabelValueLength = 63

var invalidLabelValueChars = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// NameLabelValue turns a user-facing service name into a valid label value for
// NameLabel.
//
// Characters outside [A-Za-z0-9_.-] become "-", then the result is cut to 63
// characters, then finally stripped of leading/trailing non-alphanumerics.
func NameLabelValue(name string) string {
	v := invalidLabelValueChars.ReplaceAllString(name, "-")
	if len(v) > maxLabelValueLength {
		v = v[:maxLabelValueLength]
	}
	// After the replacement, "-", "_" and "." are the only non-alphanumerics left.
	return strings.Trim(v, "-_.")
}
