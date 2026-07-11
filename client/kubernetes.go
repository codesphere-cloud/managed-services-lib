// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/codesphere-cloud/managed-services-lib/model"
)

var (
	// ErrResourceNotFound is returned when a resource is not found.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrResourceConflict is returned when a resource already exists.
	ErrResourceConflict = errors.New("resource conflict")

	// ErrResourceInvalid is returned when a resource is invalid.
	ErrResourceInvalid = errors.New("resource invalid")

	// ErrKubernetesRequestFailed is returned when a Kubernetes API request fails.
	ErrKubernetesRequestFailed = errors.New("kubernetes request failed")
)

// KubernetesClient defines the interface for Kubernetes operations.
//
//go:generate mockery --name=KubernetesClient --output=./mocks --outpkg=mocks
type KubernetesClient interface {
	// Create creates a new resource.
	Create(ctx context.Context, api model.APIResource, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)

	// Get retrieves a resource by reference.
	Get(ctx context.Context, ref model.ResourceRef) (*unstructured.Unstructured, error)

	// List lists resources matching the options.
	List(ctx context.Context, opts model.ListOptions) (*unstructured.UnstructuredList, error)

	// Patch patches a resource with JSON patches.
	Patch(ctx context.Context, ref model.ResourceRef, patches []model.K8sPatch) error

	// Delete deletes a resource by reference.
	Delete(ctx context.Context, ref model.ResourceRef) error

	// GetPodLogs retrieves logs from a pod.
	GetPodLogs(ctx context.Context, namespace, podName, container string, tailLines int64) (string, error)

	// CreateSecret creates a Kubernetes secret.
	CreateSecret(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error)

	// GetSecret retrieves a Kubernetes secret.
	GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error)

	// DeleteSecret deletes a Kubernetes secret.
	DeleteSecret(ctx context.Context, namespace, name string) error
}

// kubernetesClientImpl implements the KubernetesClient interface.
type kubernetesClientImpl struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewKubernetesClient creates a new Kubernetes client.
// Configuration is resolved in the following order:
// 1. If kubeconfig path is explicitly provided, use it
// 2. Otherwise, use default loading rules (KUBECONFIG env var, then ~/.kube/config)
// 3. If no kubeconfig is found, fall back to in-cluster configuration
func NewRESTConfig(kubeconfig string) (*rest.Config, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		// Explicit kubeconfig path provided
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Try default kubeconfig loading rules first (KUBECONFIG env, ~/.kube/config)
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()

		if err != nil {
			// Fall back to in-cluster config
			config, err = rest.InClusterConfig()
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}
	return config, nil
}

// NewKubernetesClient creates a new Kubernetes client.
func NewKubernetesClient(kubeconfig string) (KubernetesClient, error) {
	config, err := NewRESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return NewKubernetesClientFromConfig(config)
}

// NewKubernetesClientFromConfig creates a new Kubernetes client from a REST config.
func NewKubernetesClientFromConfig(config *rest.Config) (KubernetesClient, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &kubernetesClientImpl{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}, nil
}

func (c *kubernetesClientImpl) Create(ctx context.Context, api model.APIResource, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvr := toGVR(api)
	var result *unstructured.Unstructured
	var err error

	if namespace != "" {
		result, err = c.dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	} else {
		result, err = c.dynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	}

	if err != nil {
		return nil, wrapK8sError(err)
	}
	return result, nil
}

func (c *kubernetesClientImpl) Get(ctx context.Context, ref model.ResourceRef) (*unstructured.Unstructured, error) {
	gvr := toGVR(ref.APIResource)
	var result *unstructured.Unstructured
	var err error

	if ref.Namespace != "" {
		result, err = c.dynamicClient.Resource(gvr).Namespace(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	} else {
		result, err = c.dynamicClient.Resource(gvr).Get(ctx, ref.Name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, wrapK8sError(err)
	}
	return result, nil
}

func (c *kubernetesClientImpl) List(ctx context.Context, opts model.ListOptions) (*unstructured.UnstructuredList, error) {
	gvr := toGVR(opts.APIResource)
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
	}

	var result *unstructured.UnstructuredList
	var err error

	if opts.Namespace != "" {
		result, err = c.dynamicClient.Resource(gvr).Namespace(opts.Namespace).List(ctx, listOpts)
	} else {
		result, err = c.dynamicClient.Resource(gvr).List(ctx, listOpts)
	}

	if err != nil {
		return nil, wrapK8sError(err)
	}
	return result, nil
}

func (c *kubernetesClientImpl) Patch(ctx context.Context, ref model.ResourceRef, patches []model.K8sPatch) error {
	gvr := toGVR(ref.APIResource)
	patchData, err := json.Marshal(patches)
	if err != nil {
		return fmt.Errorf("failed to marshal patches: %w", err)
	}

	if ref.Namespace != "" {
		_, err = c.dynamicClient.Resource(gvr).Namespace(ref.Namespace).Patch(ctx, ref.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	} else {
		_, err = c.dynamicClient.Resource(gvr).Patch(ctx, ref.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	}

	return wrapK8sError(err)
}

func (c *kubernetesClientImpl) Delete(ctx context.Context, ref model.ResourceRef) error {
	gvr := toGVR(ref.APIResource)
	var err error

	if ref.Namespace != "" {
		err = c.dynamicClient.Resource(gvr).Namespace(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
	} else {
		err = c.dynamicClient.Resource(gvr).Delete(ctx, ref.Name, metav1.DeleteOptions{})
	}

	return wrapK8sError(err)
}

func (c *kubernetesClientImpl) GetPodLogs(ctx context.Context, namespace, podName, container string, tailLines int64) (string, error) {
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
	})

	result, err := req.DoRaw(ctx)
	if err != nil {
		return "", wrapK8sError(err)
	}
	return string(result), nil
}

func (c *kubernetesClientImpl) CreateSecret(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	result, err := c.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return nil, wrapK8sError(err)
	}
	return result, nil
}

func (c *kubernetesClientImpl) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	result, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, wrapK8sError(err)
	}
	return result, nil
}

func (c *kubernetesClientImpl) DeleteSecret(ctx context.Context, namespace, name string) error {
	err := c.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return wrapK8sError(err)
}

func toGVR(api model.APIResource) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    api.Group,
		Version:  api.Version,
		Resource: api.Plural,
	}
}

func wrapK8sError(err error) error {
	if err == nil {
		return nil
	}

	if k8serrors.IsNotFound(err) {
		return fmt.Errorf("%w: %v", ErrResourceNotFound, err)
	}
	if k8serrors.IsAlreadyExists(err) || k8serrors.IsConflict(err) {
		return fmt.Errorf("%w: %v", ErrResourceConflict, err)
	}
	if k8serrors.IsInvalid(err) {
		return fmt.Errorf("%w: %v", ErrResourceInvalid, err)
	}
	return fmt.Errorf("%w: %v", ErrKubernetesRequestFailed, err)
}
