package vllm

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubernetes client helper functions

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
