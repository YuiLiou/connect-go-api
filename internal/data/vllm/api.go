package vllm

import (
	domain "connect-go/internal/core/vllm"
	"fmt"
	"os"
	"os/exec"

	"go.yaml.in/yaml/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type VLLMAPI struct {
	Endpoint   string
	LLMDClient *LLMDClient // llm-d infra model service client
}

func NewVLLMAPI(endpoint string, llmdClient *LLMDClient) *VLLMAPI {
	return &VLLMAPI{
		Endpoint:   endpoint,
		LLMDClient: llmdClient,
	}
}

// Start creates or updates a vLLM resource using llm-d infra model service.
func (a *VLLMAPI) Start(namespace, model string) error {
	// Load YAML configuration
	obj, err := loadAndValidateYAML(model)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// Extract configuration from YAML
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("failed to extract spec from YAML: %w", err)
	}

	// Extract fields from spec
	runtimeName, _ := spec["runtimeName"].(string)
	if runtimeName == "" {
		runtimeName = model
	}

	replicas := 1
	if r, ok := spec["replicas"].(int64); ok {
		replicas = int(r)
	}

	args := []string{}
	if argsRaw, ok := spec["args"].([]interface{}); ok {
		for _, arg := range argsRaw {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	}

	storageURI := ""
	if uri, ok := spec["storageUri"].(string); ok {
		storageURI = uri
	}

	vllmConfig, _ := spec["vllmConfig"].(map[string]interface{})
	deployConfig, _ := spec["deploymentConfig"].(map[string]interface{})

	// Use llm-d infra model service to start the model
	req := &ModelStartRequest{
		Namespace:    namespace,
		RuntimeName:  runtimeName,
		Model:        model,
		Replicas:     replicas,
		Args:         args,
		StorageURI:   storageURI,
		VLLMConfig:   vllmConfig,
		DeployConfig: deployConfig,
	}

	resp, err := a.LLMDClient.StartModel(req)
	if err != nil {
		return fmt.Errorf("failed to start model via llm-d infra: %w", err)
	}

	fmt.Printf("✅ Started VLLM model via llm-d infra: %s (namespace: %s, status: %s)\n",
		model, namespace, resp.Status)
	return nil
}

// Stop stops a vLLM model using llm-d infra model service.
func (a *VLLMAPI) Stop(namespace, model string) error {
	// Load YAML configuration to get runtime name
	obj, err := loadAndValidateYAML(model)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("failed to extract spec from YAML: %w", err)
	}

	runtimeName, _ := spec["runtimeName"].(string)
	if runtimeName == "" {
		runtimeName = model
	}

	// Use llm-d infra model service to stop the model
	req := &ModelStopRequest{
		Namespace:   namespace,
		RuntimeName: runtimeName,
		Model:       model,
	}

	resp, err := a.LLMDClient.StopModel(req)
	if err != nil {
		return fmt.Errorf("failed to stop model via llm-d infra: %w", err)
	}

	fmt.Printf("✅ Stopped VLLM model via llm-d infra: %s (namespace: %s, status: %s)\n",
		model, namespace, resp.Status)
	return nil
}

// Get lists all vLLM models in the namespace using llm-d infra model service.
func (a *VLLMAPI) Get(namespace string) ([]domain.VLLMResource, error) {
	if namespace == "" {
		namespace = "default"
	}

	// Use llm-d infra model service to list models
	resp, err := a.LLMDClient.ListModels(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list models via llm-d infra: %w", err)
	}

	var resources []domain.VLLMResource
	for _, model := range resp.Models {
		// Only include running models
		if model.Phase == "Running" || model.Phase == "Ready" {
			resources = append(resources, domain.VLLMResource{
				Name:  model.Name,
				Model: model.Model,
				Phase: model.Phase,
			})
		}
	}

	if len(resources) == 0 {
		fmt.Println("No running vLLM resources found via llm-d infra.")
	}

	return resources, nil
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
