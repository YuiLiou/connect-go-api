package vllm

import (
	domain "connect-go/internal/domain/vllm"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type VLLMAPI struct {
	Endpoint string
}

var vllmGVR = schema.GroupVersionResource{
	Group:    "vllm.ai",
	Version:  "v1",
	Resource: "vllms",
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

func (a *VLLMAPI) Create(namespace, model, runtimeName string) error {
	// Define parameters
	name := "llm-runtime-mistral"
	storageUri := "file:///usr/local/models/Mixtral-8x7B-Instruct-v0.1"
	replicas := 1
	yamlFile := fmt.Sprintf("%s.yaml", name)

	// Create the CR struct
	cr := domain.VLLMCR{
		APIVersion: "vllm.ai/v1",
		Kind:       "VLLM",
		Metadata: map[string]string{
			"name":      name,
			"namespace": namespace,
		},
		Spec: map[string]interface{}{
			"namespace":   namespace,
			"model":       model,
			"runtimeName": runtimeName,
			"replicas":    replicas,
			"args": []string{
				"--disable-log-requests",
				"--max-model-len=4096",
				"--dtype=bfloat16",
			},
			"storageUri": storageUri,
			"vllmConfig": map[string]interface{}{
				"port": 8000,
				"v1":   true,
				"env": []map[string]string{
					{"name": "HF_HOME", "value": "/data"},
				},
			},
		},
	}

	// Marshal to YAML and write to file
	yamlBytes, err := yaml.Marshal(cr)
	if err != nil {
		fmt.Println("Failed to marshal YAML:", err)
		return err
	}

	if err := os.WriteFile(yamlFile, yamlBytes, 0644); err != nil {
		fmt.Println("Failed to write YAML file:", err)
		return err
	}

	fmt.Printf("VLLM CR YAML generated: %s\n", yamlFile)

	// exec kubectl apply -f <yamlFile>
	cmd := exec.Command("kubectl", "apply", "-f", yamlFile)

	// keep current environment variables, e.g., KUBECONFIG
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error applying YAML: %v\n", err)
		fmt.Printf("kubectl output:\n%s\n", string(output))
		return err
	}

	fmt.Printf("kubectl apply output:\n%s\n", string(output))
	return nil
}
