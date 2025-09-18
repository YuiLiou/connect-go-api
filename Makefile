.PHONY: proto build run

proto:
	protoc \
		--proto_path=./proto \
		--go_out=. \
		--go_opt=module=connect-go \
		--go-grpc_out=. \
		--go-grpc_opt=module=connect-go \
		--connect-go_out=. \
		--connect-go_opt=module=connect-go \
		proto/vllm/v1/vllm.proto \
		proto/greet/v1/greet.proto

build:
	go build ./cmd/server/main.go

run:
	go run ./cmd/server/main.go