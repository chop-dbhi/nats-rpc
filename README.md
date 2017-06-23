# NATS-RPC Generator

`go generate` command for creating a client interface, CLI, and serve function for a service interface.

## Install

This requires the [Protobuf Compiler](https://developers.google.com/protocol-buffers/) to be installed.

Install the library and protobuf plugins.

```
go get github.com/chop-dbhi/nats-rpc/...
go get github.com/chop-dbhi/nats-rpc/cmd/nats-rpc

# Optional for generating a CLI.
go get github.com/chop-dbhi/nats-rpc/cmd/nats-rpc-cli
```

## Usage

Create a protobuf file with a service definition.

```proto
syntax = "proto3";

package example;

message Req {
  int32 left = 1;
  int32 right = 2;
}

message Rep {
  int32 sum = 1;
}

service Service {
  rpc Sum (Req) returns (Rep);
}
```

Then run:

```
protoc --go_out=. service.proto
protoc --plugin=protoc-gen-custom="$GOPATH/bin/nats-rpc" --custom_out=. service.proto
# Optional for generating a CLI.
protoc --plugin=protoc-gen-custom="$GOPATH/bin/nats-rpc-cli" --custom_out=cmd/cli service.proto
```

See the [example](./example) package for the full example and generated output.

`service.go` contains the implementation of `Service` and `cmd/service/main.go` contains the executable code to run.

With a NATS server running on 127.0.0.1:4222, in one terminal run the server.

```
go run ./cmd/service/main.go
```

In the other, try the CLI:

```
go run ./cmd/cli/main.go Sum '{"left": 5, "right": 10}'
{"sum":15}
```

## License

MIT
