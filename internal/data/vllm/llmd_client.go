package vllm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMDClient handles communication with llm-d infra model service
type LLMDClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewLLMDClient creates a new llm-d infra client
func NewLLMDClient(baseURL string) *LLMDClient {
	return &LLMDClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ModelStartRequest represents the request to start a model
type ModelStartRequest struct {
	Namespace    string                 `json:"namespace"`
	RuntimeName  string                 `json:"runtimeName"`
	Model        string                 `json:"model"`
	Replicas     int                    `json:"replicas,omitempty"`
	Args         []string               `json:"args,omitempty"`
	StorageURI   string                 `json:"storageUri,omitempty"`
	VLLMConfig   map[string]interface{} `json:"vllmConfig,omitempty"`
	DeployConfig map[string]interface{} `json:"deploymentConfig,omitempty"`
}

// ModelStopRequest represents the request to stop a model
type ModelStopRequest struct {
	Namespace   string `json:"namespace"`
	RuntimeName string `json:"runtimeName"`
	Model       string `json:"model"`
}

// ModelStatusResponse represents the model status response
type ModelStatusResponse struct {
	Status    string `json:"status"`
	Phase     string `json:"phase"`
	Message   string `json:"message"`
	Model     string `json:"model"`
	Namespace string `json:"namespace"`
}

// ModelListResponse represents the list of models
type ModelListResponse struct {
	Models []struct {
		Name      string `json:"name"`
		Model     string `json:"model"`
		Phase     string `json:"phase"`
		Namespace string `json:"namespace"`
	} `json:"models"`
}

// StartModel sends a request to llm-d infra to start a model
func (c *LLMDClient) StartModel(req *ModelStartRequest) (*ModelStatusResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/models/start", c.BaseURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("model service returned error (status %d): %s", resp.StatusCode, string(body))
	}

	var statusResp ModelStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &statusResp, nil
}

// StopModel sends a request to llm-d infra to stop a model
func (c *LLMDClient) StopModel(req *ModelStopRequest) (*ModelStatusResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/models/stop", c.BaseURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model service returned error (status %d): %s", resp.StatusCode, string(body))
	}

	var statusResp ModelStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &statusResp, nil
}

// GetModelStatus gets the status of a specific model
func (c *LLMDClient) GetModelStatus(namespace, model string) (*ModelStatusResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/models/status?namespace=%s&model=%s", c.BaseURL, namespace, model)

	resp, err := c.HTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model service returned error (status %d): %s", resp.StatusCode, string(body))
	}

	var statusResp ModelStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &statusResp, nil
}

// ListModels lists all models in a namespace
func (c *LLMDClient) ListModels(namespace string) (*ModelListResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/models/list?namespace=%s", c.BaseURL, namespace)

	resp, err := c.HTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model service returned error (status %d): %s", resp.StatusCode, string(body))
	}

	var listResp ModelListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &listResp, nil
}
