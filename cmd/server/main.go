package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	greetv1 "connect-go/api/greetv1"
	greetv1connect "connect-go/api/greetv1/greetv1connect"
	"io"
)

type GreetServer struct{}

func (s *GreetServer) Greet(
	ctx context.Context,
	req *connect.Request[greetv1.GreetRequest],
) (*connect.Response[greetv1.GreetResponse], error) {
	log.Println("Request headers: ", req.Header())
	res := connect.NewResponse(&greetv1.GreetResponse{
		Greeting: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	res.Header().Set("Greet-Version", "v1")
	return res, nil
}

func updateVLLMStatusHandler(w http.ResponseWriter, r *http.Request) {
	type UpdateRequest struct {
		Model  string `json:"model"`
		Action string `json:"action"` // start, stop, update
	}
	var req UpdateRequest
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &req)

	// 根據 action 呼叫 vLLM API
	var vllmAPI string
	switch req.Action {
	case "start":
		vllmAPI = "http://vllm-service:8000/v1/start"
	case "stop":
		vllmAPI = "http://vllm-service:8000/v1/stop"
	case "update":
		vllmAPI = "http://vllm-service:8000/v1/update"
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// 假設 vLLM API 需要 model 參數
	payload, _ := json.Marshal(map[string]string{"model": req.Model})
	resp, err := http.Post(vllmAPI, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		http.Error(w, "Failed to update vLLM", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte("vLLM status updated"))
}

func main() {
	log.Println("Starting server on localhost:8799")
	greeter := &GreetServer{}
	mux := http.NewServeMux()
	path, handler := greetv1connect.NewGreetServiceHandler(greeter)
	log.Println("Registering handler for path: ", path)
	mux.Handle(path, handler)
	mux.HandleFunc("/vllm/update", updateVLLMStatusHandler)
	err := http.ListenAndServe(
		"localhost:8799",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
