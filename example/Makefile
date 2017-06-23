proto:
	protoc --go_out=. service.proto
	protoc --plugin=protoc-gen-custom=$(GOPATH)/bin/nats-rpc --custom_out=. service.proto
	protoc --plugin=protoc-gen-custom=$(GOPATH)/bin/nats-rpc-cli --custom_out=cmd/cli service.proto
