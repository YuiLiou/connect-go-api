package vllm

import (
	domain "connect-go/internal/core/vllm"
	vllmv1 "connect-go/api/vllmv1"
	"connect-go/api/vllmv1/vllmv1connect"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"connectrpc.com/connect"
	"go.yaml.in/yaml/v2"
)


type VLLMAPI struct {
	Endpoint      string
	ModelServiceClient vllmv1connect.LLMApiServiceClient
}

func NewVLLMAPI(endpoint string, modelServiceEndpoint string) *VLLMAPI {
	// Create HTTP client for model service
	httpClient := &http.Client{}
	modelServiceClient := vllmv1connect.NewLLMApiServiceClient(httpClient, modelServiceEndpoint)
	
	return &VLLMAPI{
		Endpoint: endpoint,
		ModelServiceClient: modelServiceClient,
	}
}

// Start starts a model using the llm-d model service infrastructure.
func (a *VLLMAPI) Start(namespace, model string) error {
	if namespace == "" {
		namespace = "default"
	}

	// Call the model service to start the LLM
	ctx := context.Background()
	req := connect.NewRequest(&vllmv1.LLMRequest{
		Namespace:   namespace,
		RuntimeName: model,
	})

	resp, err := a.ModelServiceClient.StartLLM(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start LLM via model service: %w", err)
	}

	fmt.Printf("Successfully started LLM: %s\n", resp.Msg.Message)
	return nil
}

// Stop stops a model using the llm-d model service infrastructure.
func (a *VLLMAPI) Stop(namespace, model string) error {
	if namespace == "" {
		namespace = "default"
	}

	// Call the model service to stop the LLM
	ctx := context.Background()
	req := connect.NewRequest(&vllmv1.LLMRequest{
		Namespace:   namespace,
		RuntimeName: model,
	})

	resp, err := a.ModelServiceClient.StopLLM(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to stop LLM via model service: %w", err)
	}

	fmt.Printf("Successfully stopped LLM: %s\n", resp.Msg.Message)
	return nil
}

// Get lists all LLMs using the llm-d model service infrastructure.
func (a *VLLMAPI) Get(namespace string) ([]domain.VLLMResource, error) {
	if namespace == "" {
		namespace = "default"
	}

	// Call the model service to list LLMs
	ctx := context.Background()
	req := connect.NewRequest(&vllmv1.ListLLMsRequest{
		Namespace: namespace,
	})

	resp, err := a.ModelServiceClient.ListLLMs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list LLMs via model service: %w", err)
	}

	// Convert response to domain.VLLMResource
	var resources []domain.VLLMResource
	for _, llmInfo := range resp.Msg.Llms {
		// Extract status phase from status map
		// For now, use a simple status representation
		// The actual phase would be in llmInfo.Status["phase"] as a protobuf Any
		phase := "Running" // Default phase, model service should provide this
		
		resources = append(resources, domain.VLLMResource{
			Name:  llmInfo.Name,
			Model: llmInfo.Model,
			Phase: phase,
		})
	}

	if len(resources) == 0 {
		fmt.Println("No LLM resources found.")
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
