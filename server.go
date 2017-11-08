package natsrpc

import (
	"context"

	"github.com/chop-dbhi/nats-rpc/transport"
)

type Server interface {
	Serve(context.Context, ...transport.SubscribeOption) error
}
