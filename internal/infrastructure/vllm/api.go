package vllm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

type VLLMAPI struct {
	Endpoint string
}

var vllmGVR = schema.GroupVersionResource{
	Group:    "vllm.ai",
	Version:  "v1",
	Resource: "vllms",
}

// getDynamicClient creates a dynamic Kubernetes client using the kubeconfig.
func (a *VLLMAPI) getDynamicClient() (dynamic.Interface, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}
	return dynamic.NewForConfig(config)
}

// loadAndValidateYAML loads and decodes the model-specific YAML file, validating the Kind.
func loadAndValidateYAML(model string) (*unstructured.Unstructured, error) {
	if model == "" {
		return nil, fmt.Errorf("model name is required")
	}
	yamlFileName := fmt.Sprintf("config/samples/%s.yaml", model)
	yamlFile, err := os.ReadFile(yamlFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file %q: %w", yamlFileName, err)
	}
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decoder.Decode(yamlFile, nil, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}
	if gvk.Kind != "VLLM" {
		return nil, fmt.Errorf("expected Kind=VLLM, got %s", gvk.Kind)
	}
	return obj, nil
}

// Start creates or updates a vLLM resource in Kubernetes to initiate the start action.
func (a *VLLMAPI) Start(model string) error {
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

	ctx := context.Background()
	namespace := "default" // Consider making this configurable.
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
func (a *VLLMAPI) Stop(model string) error {
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

	ctx := context.Background()
	namespace := "default" // Consider making this configurable.
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

func (a *VLLMAPI) Update(model string) error {
	return nil
}
