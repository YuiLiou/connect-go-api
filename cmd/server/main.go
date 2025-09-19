package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	greetv1 "connect-go/api/greetv1"
	greetv1connect "connect-go/api/greetv1/greetv1connect"
	vllmInfra "connect-go/internal/infrastructure/vllm"
	vllmIface "connect-go/internal/interfaces/vllm"
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

func main() {
	log.Println("Starting server on localhost:8799")
	greeter := &GreetServer{}
	mux := http.NewServeMux()
	path, handler := greetv1connect.NewGreetServiceHandler(greeter)
	log.Println("Registering handler for path: ", path)
	mux.Handle(path, handler)
	vllmAPI := &vllmInfra.VLLMAPI{Endpoint: "http://vllm-service:8000"}
	vllmHandler := &vllmIface.VLLMHandler{Service: vllmAPI}
	mux.HandleFunc("/vllm/update", vllmHandler.UpdateVLLMStatus)
	err := http.ListenAndServe(
		"localhost:8799",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
