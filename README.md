# NATS-RPC Generator

`go generate` command for creating a client interface, CLI, and serve function for a service interface.

## Install

This requires the [Protobuf Compiler](https://developers.google.com/protocol-buffers/) to be installed.

Install the library and protobuf plugins.

```
go get github.com/chop-dbhi/nats-rpc/...
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
protoc \
  --go_out=. \
  --nats-rpc_out=. \
  --nats-rpc-cli_out=cmd/cli \
  service.proto
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

### Parameters

Two parameters are supported for both commands. Parameters are supplied as a set of param-value pairs separated by commas as shown below.

```
protoc --nats-rpc_out=param1=value1,param2=value2:. service.proto
```

`subject` - A Go template string that enables customizing the generated NATS subject prefix. Supported template parameters include:
  - `{{.Pkg}}` - The name of the package defined in the proto file.
  - `{{.Service}}` - The name of the service type being generated for.

A subject is produced per method defined on the service and will be appended to this subject prefix in the code. The default subject prefix template is `{{.Pkg}}`.

`outfile` - The name of the output file.

## License

MIT
