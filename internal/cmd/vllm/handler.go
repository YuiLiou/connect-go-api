package vllm

import (
	"connect-go/internal/app/vllm"
	domain "connect-go/internal/core/vllm"
	"encoding/json"
	"net/http"
)

type SwitchRequest struct {
	Namespace   string `json:"namespace"`
	RuntimeName string `json:"runtimeName"`
	Model       string `json:"model"`
}

type GetRequest struct {
	Namespace string `json:"namespace"`
}

type VLLMHandler struct {
	Service vllm.VLLMService
}

func NewVLLMHandler(service vllm.VLLMService) *VLLMHandler {
	return &VLLMHandler{Service: service}
}

func (h *VLLMHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req SwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Namespace == "" || req.RuntimeName == "" || req.Model == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}
	vllm, err := h.Service.Start(req.Namespace, req.RuntimeName, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.writeResponse(w, req, vllm.Status, "vLLM started")
}

func (h *VLLMHandler) Stop(w http.ResponseWriter, r *http.Request) {
	var req SwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Namespace == "" || req.RuntimeName == "" || req.Model == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}
	vllm, err := h.Service.Stop(req.Namespace, req.RuntimeName, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.writeResponse(w, req, vllm.Status, "vLLM stopped")
}

func (h *VLLMHandler) Get(w http.ResponseWriter, r *http.Request) {
	var req GetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Namespace == "" {
		http.Error(w, "Namespace is required", http.StatusBadRequest)
		return
	}
	vllms, err := h.Service.Get(req.Namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	statuses := make([]string, len(vllms))
	models := make([]string, len(vllms))
	runtimeNames := make([]string, len(vllms))
	for i, v := range vllms {
		statuses[i] = string(v.Phase)
		models[i] = v.Model
		runtimeNames[i] = v.Name
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Message      string   `json:"message"`
		Namespace    string   `json:"namespace"`
		RuntimeNames []string `json:"runtimeNames"`
		Models       []string `json:"model"`
		Statuses     []string `json:"statuses"`
	}{
		Message:      "vLLM updated",
		Namespace:    req.Namespace,
		RuntimeNames: runtimeNames,
		Models:       models,
		Statuses:     statuses,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *VLLMHandler) writeResponse(w http.ResponseWriter, req SwitchRequest, status domain.Status, message string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Message     string `json:"message"`
		Namespace   string `json:"namespace"`
		RuntimeName string `json:"runtimeName"`
		Model       string `json:"model"`
		Status      string `json:"status"`
	}{
		Message:     message,
		Namespace:   req.Namespace,
		RuntimeName: req.RuntimeName,
		Model:       req.Model,
		Status:      string(status),
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
