proto:
	protoc \
	--go_out=. \
	--go_opt=module=connect-go \
	--go-grpc_out=. \
	--go-grpc_opt=module=connect-go \
	--connect-go_out=. \
	--connect-go_opt=module=connect-go \
	greet/v1/greet.proto

build:
	go build ./cmd/server/main.go

run:
	go run ./cmd/server/main.go