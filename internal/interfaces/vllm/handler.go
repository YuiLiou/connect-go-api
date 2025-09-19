package vllm

import (
	"connect-go/internal/application/vllm"
	"encoding/json"
	"net/http"
)

type UpdateRequest struct {
	Model  string `json:"model"`
	Action string `json:"action"`
}

type VLLMHandler struct {
	Service vllm.VLLMService
}

func (h *VLLMHandler) UpdateVLLMStatus(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	var err error
	switch req.Action {
	case "start":
		err = h.Service.Start(req.Model)
	case "stop":
		err = h.Service.Stop(req.Model)
	case "update":
		err = h.Service.Update(req.Model)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("vLLM status updated"))
}
