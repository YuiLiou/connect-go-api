package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	greetv1 "connect-go/api/greetv1"
	greetv1connect "connect-go/api/greetv1/greetv1connect"
	vllmApp "connect-go/internal/application/vllm"
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
	vllmAPIEndpoint := os.Getenv("VLLM_API_ENDPOINT")
	if vllmAPIEndpoint == "" {
		vllmAPIEndpoint = "http://vllm-service:8000"
	}

	vllmAPI := &vllmInfra.VLLMAPI{Endpoint: vllmAPIEndpoint}
	vllmRepo := vllmInfra.NewInMemoryVLLMRepository()
	vllmService := vllmApp.NewVLLMServiceImpl(vllmAPI, vllmRepo)
	vllmHandler := vllmIface.NewVLLMHandler(vllmService)

	mux := http.NewServeMux()
	greeter := &GreetServer{}
	path, handler := greetv1connect.NewGreetServiceHandler(greeter)
	log.Println("Registering gRPC handler for path: ", path)
	mux.Handle(path, handler)

	mux.HandleFunc("/v1/vllm/start", vllmHandler.Start)
	mux.HandleFunc("/v1/vllm/stop", vllmHandler.Stop)
	mux.HandleFunc("/v1/vllm/update", vllmHandler.Update)

	server := &http.Server{
		Addr:    "localhost:8799",
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	log.Println("Starting server on localhost:8799")
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// 優雅關閉（簡化示例，實際應處理信號）
	time.Sleep(time.Hour) // 模擬運行
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
