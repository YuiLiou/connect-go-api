package vllm

import (
	domain "connect-go/internal/core/vllm"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"go.yaml.in/yaml/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var vllmGVR = schema.GroupVersionResource{
	Group:    "vllm.ai",
	Version:  "v1",
	Resource: "vllms",
}

type VLLMAPI struct {
	Endpoint string
}

func NewVLLMAPI(endpoint string) *VLLMAPI {
	return &VLLMAPI{
		Endpoint: endpoint,
	}
}

// Start creates or updates a vLLM resource in Kubernetes to initiate the start action.
func (a *VLLMAPI) Start(namespace, model string) error {
	obj, err := loadAndValidateYAML(model)
	if err != nil {
		return err
	}
	resourceName := obj.GetName()
	if resourceName == "" {
		return fmt.Errorf("YAML must specify metadata.name for the resource")
	}

	// Override model and runtimeName if necessary.
	modelInYaml, found, err := unstructured.NestedString(obj.Object, "spec", "model")
	if err != nil {
		return fmt.Errorf("failed to get spec.model from YAML: %w", err)
	}
	if !found || model != modelInYaml {
		if !found {
			fmt.Printf("spec.model not found in YAML; setting to %s\n", model)
		} else {
			fmt.Printf("Overriding spec.model from %s to %s\n", modelInYaml, model)
		}
		if err := unstructured.SetNestedField(obj.Object, model, "spec", "model"); err != nil {
			return fmt.Errorf("failed to set spec.model: %w", err)
		}
		if err := unstructured.SetNestedField(obj.Object, model, "spec", "runtimeName"); err != nil {
			return fmt.Errorf("failed to set spec.runtimeName: %w", err)
		}
	}

	// Set action to "start" in the object (for creation or as base for patch).
	if err := unstructured.SetNestedField(obj.Object, "start", "spec", "action"); err != nil {
		return fmt.Errorf("failed to set spec.action: %w", err)
	}

	dynamicClient, err := a.getDynamicClient()
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	ctx := context.Background()
	resourceClient := dynamicClient.Resource(vllmGVR).Namespace(namespace)

	// Check if the resource exists.
	_, err = resourceClient.Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get VLLM resource %q: %w", resourceName, err)
		}
		// Resource does not exist: create it.
		_, err = resourceClient.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create VLLM resource %q: %w", resourceName, err)
		}
		fmt.Printf("Created VLLM resource in Kubernetes: %s (model: %s)\n", resourceName, model)
		return nil
	}

	// Resource exists: patch spec.action to "start" (and model/runtimeName if overridden).
	updatedSpec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("failed to extract spec from YAML: %w", err)
	}

	// Create merge patch for spec.
	patch := map[string]interface{}{
		"spec": updatedSpec,
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	// Apply patch.
	_, err = resourceClient.Patch(ctx, resourceName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch VLLM resource %q: %w", resourceName, err)
	}

	fmt.Printf("Updated VLLM resource in Kubernetes: %s (model: %s, action: start)\n", resourceName, model)
	return nil
}

// Stop updates an existing vLLM resource in Kubernetes to initiate the stop action.
func (a *VLLMAPI) Stop(namespace, model string) error {
	obj, err := loadAndValidateYAML(model)
	if err != nil {
		return err
	}
	resourceName := obj.GetName()
	if resourceName == "" {
		return fmt.Errorf("YAML must specify metadata.name for the resource")
	}

	dynamicClient, err := a.getDynamicClient()
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	ctx := context.Background()
	resourceClient := dynamicClient.Resource(vllmGVR).Namespace(namespace)

	// Check if the resource exists.
	_, err = resourceClient.Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("VLLM resource %q not found; cannot stop", resourceName)
		}
		return fmt.Errorf("failed to get VLLM resource %q: %w", resourceName, err)
	}

	// Create merge patch for spec.action.
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"action": "stop",
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	// Apply patch.
	_, err = resourceClient.Patch(ctx, resourceName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch VLLM resource %q: %w", resourceName, err)
	}

	fmt.Printf("Updated VLLM resource in Kubernetes to stop: %s (model: %s)\n", resourceName, model)
	return nil
}

// Get lists and displays all vLLM custom resources in the namespace that are currently in the "Running" status phase.
func (a *VLLMAPI) Get(namespace string) ([]domain.VLLMResource, error) {
	if namespace == "" {
		namespace = "default"
	}

	dynamicClient, err := a.getDynamicClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resourceClient := dynamicClient.Resource(vllmGVR).Namespace(namespace)

	// List all vLLM resources.
	list, err := resourceClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list VLLM resources: %w", err)
	}

	var runningResources []domain.VLLMResource
	for _, item := range list.Items {
		phase, foundPhase, err := unstructured.NestedString(item.Object, "status", "phase")
		if err != nil || !foundPhase || phase != "Running" {
			continue
		}

		model, foundModel, err := unstructured.NestedString(item.Object, "spec", "model")
		if err != nil || !foundModel {
			model = "unknown"
		}

		resourceName := item.GetName()
		runningResources = append(runningResources, domain.VLLMResource{
			Name:  resourceName,
			Model: model,
			Phase: phase,
		})
	}
	if len(runningResources) == 0 {
		fmt.Println("No running vLLM resources found.")
	}

	return runningResources, nil
}

type CreateParams struct {
	Namespace              string
	Name                   string
	Model                  string
	RuntimeName            string
	StorageUri             string
	DeviceIDs              []string
	GPUMemoryUtilization   float64
	MaxModelLen            int64
	TensorParallelSize     int64
	EnablePromptTokenStats bool
	Replicas               int
}

// Create and apply VLLM CR
func (a *VLLMAPI) Create(p CreateParams) error {
	// Build args
	args := buildArgs(p)

	// Build DeviceRequests
	deviceRequests := buildDeviceRequests(p.DeviceIDs)

	// CR struct
	cr := domain.VLLMCR{
		APIVersion: "vllm.ai/v1",
		Kind:       "VLLM",
		Metadata: map[string]string{
			"name":      p.Name,
			"namespace": p.Namespace,
		},
		Spec: map[string]interface{}{
			"namespace":   p.Namespace,
			"model":       p.Model,
			"runtimeName": p.RuntimeName,
			"replicas":    p.Replicas,
			"args":        args,
			"storageUri":  p.StorageUri,
			"action":      "start",
			"vllmConfig": map[string]interface{}{
				"port": 8000,
				"v1":   true,
				"env": []map[string]string{
					{"name": "HF_HOME", "value": "/data"},
				},
			},
			"deploymentConfig": map[string]interface{}{
				"resources": map[string]interface{}{
					"limits": map[string]string{
						"nvidia.com/gpu": fmt.Sprintf("%d", len(p.DeviceIDs)),
					},
					"requests": map[string]string{
						"cpu":    "10",
						"memory": "32Gi",
					},
				},
				"deviceRequests": deviceRequests,
				"image": map[string]string{
					"registry":   "docker.io",
					"name":       "lmcache/vllm-openai:2025-05-27-v1",
					"pullPolicy": "IfNotPresent",
				},
			},
		},
	}

	// Write YAML
	yamlFile := fmt.Sprintf("%s.yaml", p.Name)
	if err := writeYAML(cr, yamlFile); err != nil {
		return err
	}

	// Apply CR
	return applyYAML(yamlFile)
}

func buildArgs(p CreateParams) []string {
	args := []string{
		fmt.Sprintf("--gpu-memory-utilization=%.1f", p.GPUMemoryUtilization),
		fmt.Sprintf("--max-model-len=%d", p.MaxModelLen),
		fmt.Sprintf("--tensor-parallel-size=%d", p.TensorParallelSize),
	}
	if p.EnablePromptTokenStats {
		args = append(args, "--enable-prompt-tokens-details")
	}
	return args
}

func buildDeviceRequests(deviceIDs []string) []map[string]interface{} {
	if len(deviceIDs) == 0 {
		return nil
	}
	return []map[string]interface{}{
		{
			"driver":       "nvidia",
			"count":        len(deviceIDs),
			"capabilities": []string{"gpu", "nvidia-compute"},
			"deviceIDs":    deviceIDs,
		},
	}
}

func writeYAML(cr interface{}, file string) error {
	yamlBytes, err := yaml.Marshal(cr)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	if err := os.WriteFile(file, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}
	fmt.Printf("✅ VLLM CR YAML generated: %s\n", file)
	return nil
}

func applyYAML(file string) error {
	cmd := exec.Command("kubectl", "apply", "-f", file)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply error: %v\n%s", err, string(output))
	}
	fmt.Printf("✅ kubectl apply success:\n%s\n", string(output))
	return nil
}
