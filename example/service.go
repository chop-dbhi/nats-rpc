//go:generate nats-rpc -type=Service -client=client.go -cli=./cmd/cli/main.go
package example

import "context"

type Service interface {
	Add(context.Context, *Req) (*Rep, error)
}
