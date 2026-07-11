// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/codesphere-cloud/managed-services-lib/client"
	"github.com/codesphere-cloud/managed-services-lib/model"
)

// DefaultPausedReplicas is the replica count providers restore to on unpause when
// the stored replica count is missing or unparseable.
const DefaultPausedReplicas = 1

var (
	// ErrServiceNotFound is returned when a service is not found.
	ErrServiceNotFound = errors.New("service not found")

	// ErrBackupNotFound is returned when a backup is not found.
	ErrBackupNotFound = errors.New("backup not found")

	// ErrInvalidArgument is returned when an argument is invalid.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrServiceNotHealthy is returned when a service is not healthy.
	ErrServiceNotHealthy = errors.New("service not healthy")

	// ErrNamespaceNotFound is returned when the team namespace does not exist.
	ErrNamespaceNotFound = errors.New("namespace not found")

	ErrNotImplemented = errors.New("not implemented")
)

// KubernetesClient is an alias for the client interface used by providers.
type KubernetesClient = client.KubernetesClient

// Base provides common functionality for managed service providers.
// Concrete providers should embed this struct and implement provider-specific logic.
type Base struct {
	K8sClient KubernetesClient
	Logger    *slog.Logger
}

// NewBase creates a new Base provider.
func NewBase(k8sClient KubernetesClient, logger *slog.Logger) *Base {
	return &Base{
		K8sClient: k8sClient,
		Logger:    logger,
	}
}

// NamespaceForTeam returns the namespace for a given team ID.
func NamespaceForTeam(teamID int) string {
	return fmt.Sprintf("rg-%d", teamID)
}

// FindServiceNamespace discovers the namespace for a service ID by cluster-wide label lookup.
func (p *Base) FindServiceNamespace(ctx context.Context, api model.APIResource, providerType string, id model.ServiceID) (string, error) {
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", model.ProviderLabel, providerType, model.ServiceIDLabel, string(id))
	list, err := p.K8sClient.List(ctx, model.ListOptions{
		APIResource:   api,
		Namespace:     "", // Cluster-wide
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("failed to find service %s: %w", id, err)
	}
	if len(list.Items) == 0 {
		return "", fmt.Errorf("%w: %s", ErrServiceNotFound, id)
	}
	return list.Items[0].GetNamespace(), nil
}

// CreateSecret creates a Kubernetes secret for the managed service.
func (p *Base) CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error {
	secret := p.buildSecret(namespace, name, data, labels)
	_, err := p.K8sClient.CreateSecret(ctx, namespace, secret)
	if err != nil {
		return fmt.Errorf("failed to create secret %s: %w", name, err)
	}
	return nil
}

// DeleteSecret deletes a Kubernetes secret.
func (p *Base) DeleteSecret(ctx context.Context, namespace, name string) error {
	err := p.K8sClient.DeleteSecret(ctx, namespace, name)
	if err != nil && !errors.Is(err, client.ErrResourceNotFound) {
		return fmt.Errorf("failed to delete secret %s: %w", name, err)
	}
	return nil
}

// GetResourceRef creates a ResourceRef for the given API resource, name, and namespace.
func (p *Base) GetResourceRef(api model.APIResource, name, namespace string) model.ResourceRef {
	return model.ResourceRef{
		APIResource: api,
		Name:        name,
		Namespace:   namespace,
	}
}

// ListByLabel lists resources by label selector across all namespaces.
func (p *Base) ListByLabel(ctx context.Context, api model.APIResource, labelSelector string) ([]string, error) {
	list, err := p.K8sClient.List(ctx, model.ListOptions{
		APIResource:   api,
		Namespace:     "", // Cluster-wide
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	ids := []string{}
	for _, item := range list.Items {
		if labels := item.GetLabels(); labels != nil {
			if id, ok := labels[model.ServiceIDLabel]; ok {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

// StoredReplicas reads the replica count stored in the given annotation on the
// referenced resource, used to restore the desired count on unpause. It falls back
// to DefaultPausedReplicas when the annotation is absent or unparseable.
func (p *Base) StoredReplicas(ctx context.Context, ref model.ResourceRef, annotationKey string) (int, error) {
	obj, err := p.K8sClient.Get(ctx, ref)
	if err != nil {
		return 0, err
	}
	return ReplicasFromAnnotation(obj.GetAnnotations(), annotationKey), nil
}

// ReplicasFromAnnotation parses the replica count stored in annotations under key,
// returning DefaultPausedReplicas when the annotation is missing or not an integer.
func ReplicasFromAnnotation(annotations map[string]string, key string) int {
	replicasStr, ok := annotations[key]
	if !ok {
		return DefaultPausedReplicas
	}
	replicas, err := strconv.Atoi(replicasStr)
	if err != nil {
		return DefaultPausedReplicas
	}
	return replicas
}

// EscapeJSONPointer escapes a string for use as a single JSON Pointer reference
// token (RFC 6901): "~" becomes "~0" and "/" becomes "~1". The "~" replacement
// must happen first.
func EscapeJSONPointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// ParseSizeMiB parses a Kubernetes quantity string (e.g. "5Gi", "1024Mi") into
// whole MiB. It returns 0 when the string is empty or cannot be parsed.
func ParseSizeMiB(s string) int {
	if s == "" {
		return 0
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return 0
	}
	return int(q.Value() / (1024 * 1024))
}

func (p *Base) buildSecret(namespace, name string, data map[string][]byte, labels map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
}
