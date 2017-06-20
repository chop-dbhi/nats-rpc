# NATS-RPC Generator

`go generate` command for creating a client interface, CLI, and serve function for a service interface.

## Usage

```go
//go:generate nats-rpc -type=Service -client=client.go -cli=./cmd/cli/main.go
package main

type Req struct {
  Left int
  Right int
}

type Rep struct {
  Sum int
}

type Service interface {
  Add(context.Context, *Req) (*Rep, error)
}
```

## Options

- `type` - Name of the service interface type. All methods are expected to have the same signature. `(context.Context, <RequestType>) (<ResponseType>, error)` where the request and response types can be user-defined.
- `client` - Name of the output file to write the client type and serve function.
- `cli` - Name of the output file to write the CLI.
- `group` - Name of the NATS queue group for the serve subscription handlers. Defaults to `svc.<pkg-name>`.
- `prefix` - Prefix to all NATS subjects used. Defaults to no prefix.
