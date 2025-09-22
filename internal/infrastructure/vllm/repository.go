package vllm

import (
	"connect-go/internal/domain/vllm"
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type VLLMRepository interface {
	FindByModel(namespace, runtimeName, model string) (*vllm.VLLM, error)
	Save(vllm *vllm.VLLM) error
	UpdateCRStatusToStart(namespace, name, model string) error
}

// K8sVLLMRepository implements VLLMRepository for actual Kubernetes interactions
type K8sVLLMRepository struct {
	client kubernetes.Interface
	config *rest.Config
}

func NewK8sVLLMRepository(client kubernetes.Interface, config *rest.Config) *K8sVLLMRepository {
	return &K8sVLLMRepository{
		client: client,
		config: config,
	}
}

// Implement the interface methods for K8sVLLMRepository
func (r *K8sVLLMRepository) FindByModel(namespace, runtimeName, model string) (*vllm.VLLM, error) {
	return vllm.NewVLLM(namespace, runtimeName, model), nil
}

func (r *K8sVLLMRepository) Save(vllm *vllm.VLLM) error {
	return nil
}

// UpdateCRStatusToStart implements the actual K8s CR status update
func (r *K8sVLLMRepository) UpdateCRStatusToStart(namespace, name, model string) error {
	// Create context for the API call
	ctx := context.Background()

	// Create a status update payload
	now := metav1.Now()
	statusUpdate := map[string]interface{}{
		"status": map[string]interface{}{
			"phase":     "Starting",
			"message":   fmt.Sprintf("vLLM model '%s' is starting", model),
			"startTime": now,
			"conditions": []map[string]interface{}{
				{
					"type":               "Starting",
					"status":             "True",
					"lastTransitionTime": now,
					"reason":             "ModelStartRequested",
					"message":            "vLLM model start operation initiated",
				},
			},
		},
	}

	// Convert the update to JSON
	patchBytes, err := json.Marshal(statusUpdate)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	// Get the dynamic client for the custom resource
	dynamicClient, err := r.getDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %w", err)
	}

	// Define the GVR (GroupVersionResource) for your CR
	gvr := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	// Patch the CR status
	_, err = dynamicClient.Resource(r.getVLLMGVR()).
		Namespace(gvr.Namespace).
		Patch(ctx, gvr.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("failed to patch CR status: %w", err)
	}

	// Log the successful update
	fmt.Printf("Successfully updated CR %s in namespace %s to status: Starting for model %s\n",
		name, namespace, model)

	return nil
}

func (r *K8sVLLMRepository) getDynamicClient() (dynamic.Interface, error) {
	dynamicClient, err := dynamic.NewForConfig(r.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}
	return dynamicClient, nil
}

func (r *K8sVLLMRepository) getVLLMGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vllm.ai",
		Version:  "v1",
		Resource: "vllms",
	}
}
