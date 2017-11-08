proto:
	protoc --go_out=. service.proto
	protoc --nats-rpc_out=. service.proto
	protoc --nats-rpc-cli_out=cmd/cli service.proto
