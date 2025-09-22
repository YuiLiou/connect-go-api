package vllm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

type VLLMAPI struct {
	Endpoint string
}

func (a *VLLMAPI) callAPI(action, model string) error {
	url := fmt.Sprintf("%s/v1/%s", a.Endpoint, action)
	payload, err := json.Marshal(map[string]string{"model": model})
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vLLM API error: %s", resp.Status)
	}
	return nil
}

func (a *VLLMAPI) Start(model string) error {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	// Create a dynamic Kubernetes client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	// Define the GVR for the VLLM custom resource
	gvr := schema.GroupVersionResource{
		Group:    "vllm.ai",
		Version:  "v1",
		Resource: "vllms",
	}

	// Read the YAML file for the specified model
	yamlFileName := fmt.Sprintf("config/samples/%s.yaml", model)
	yamlFile, err := os.ReadFile(yamlFileName)
	if err != nil {
		return fmt.Errorf("Read YAML file failed: %w", err)
	}

	// Decode YAML into an unstructured object
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decoder.Decode(yamlFile, nil, obj)
	if err != nil {
		return fmt.Errorf("Decoding YAML failed: %w", err)
	}

	// Ensure we parsed the correct Kind
	if gvk.Kind != "VLLM" {
		return fmt.Errorf("gvk.Kind != VLLM, gvk = %s", gvk.Kind)
	}

	modelInYaml, found, err := unstructured.NestedString(obj.Object, "spec", "model")
	if err != nil {
		return fmt.Errorf("failed to get model from YAML: %w", err)
	}

	if !found || model != modelInYaml {
		if !found {
			fmt.Printf("Model not found in YAML, setting to %s\n", model)
		} else {
			fmt.Printf("Model in YAML (%s) does not match requested model (%s), updating\n", modelInYaml, model)
		}
		if err := unstructured.SetNestedField(obj.Object, model, "spec", "model"); err != nil {
			return fmt.Errorf("failed to set model in YAML: %w", err)
		}
		if err := unstructured.SetNestedField(obj.Object, model, "spec", "runtimeName"); err != nil {
			return fmt.Errorf("failed to set runtimeName in YAML: %w", err)
		}
	}

	// Ensure action is set to "start"
	if err := unstructured.SetNestedField(obj.Object, "start", "spec", "action"); err != nil {
		return fmt.Errorf("failed to set action in YAML: %w", err)
	}

	// Create the VLLM resource in Kubernetes
	ctx := context.Background()
	_, err = dynamicClient.Resource(gvr).Namespace("default").Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create VLLM resource: %w", err)
	}

	fmt.Printf("Created VLLM resource in Kubernetes: %s\n", model)
	return nil
}

func (a *VLLMAPI) Stop(model string) error {
	return a.callAPI("stop", model)
}

func (a *VLLMAPI) Update(model string) error {
	return a.callAPI("update", model)
}
