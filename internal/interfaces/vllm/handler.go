package vllm

import (
	"connect-go/internal/application/vllm"
	domain "connect-go/internal/domain/vllm"
	"encoding/json"
	"net/http"
)

type UpdateRequest struct {
	Model string `json:"model"`
}

type VLLMHandler struct {
	Service vllm.VLLMService
}

func NewVLLMHandler(service vllm.VLLMService) *VLLMHandler {
	return &VLLMHandler{Service: service}
}

func (h *VLLMHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		http.Error(w, "Model is required", http.StatusBadRequest)
		return
	}
	err := h.Service.Start(req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status, _ := h.Service.GetStatus(req.Model)
	h.writeResponse(w, req.Model, status, "vLLM started")
}

func (h *VLLMHandler) Stop(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		http.Error(w, "Model is required", http.StatusBadRequest)
		return
	}
	err := h.Service.Stop(req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status, _ := h.Service.GetStatus(req.Model)
	h.writeResponse(w, req.Model, status, "vLLM stopped")
}

func (h *VLLMHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		http.Error(w, "Model is required", http.StatusBadRequest)
		return
	}
	err := h.Service.Update(req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status, _ := h.Service.GetStatus(req.Model)
	h.writeResponse(w, req.Model, status, "vLLM updated")
}

func (h *VLLMHandler) writeResponse(w http.ResponseWriter, model string, status domain.Status, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
		Model   string `json:"model"`
		Status  string `json:"status"`
	}{
		Message: message,
		Model:   model,
		Status:  string(status),
	})
}
