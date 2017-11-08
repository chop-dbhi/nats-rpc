package example

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chop-dbhi/nats-rpc"
	"github.com/chop-dbhi/nats-rpc/transport"
	"github.com/golang/protobuf/proto"
)

type Service interface {
	Sum(context.Context, *Req) (*Rep, error)
}

type Client interface {
	Sum(context.Context, *Req, ...transport.RequestOption) (*Rep, error)
}

// client is an implementation of Client.
type client struct {
	tp transport.Transport
}

func (c *client) Sum(ctx context.Context, req *Req, opts ...transport.RequestOption) (*Rep, error) {
	var rep Rep

	_, err := c.tp.Request("example.Sum", req, &rep, opts...)
	if err != nil {
		return nil, err
	}

	return &rep, nil
}

// NewClient creates a new Service client.
func NewClient(tp transport.Transport) Client {
	return &client{tp}
}

type server struct {
	tp  transport.Transport
	svc Service
}

func (s *server) Serve(ctx context.Context, opts ...transport.SubscribeOption) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error
	_, err = s.tp.Subscribe("example.>", func(msg *transport.Message) (proto.Message, error) {
		switch msg.Subject {
		case "example.Sum":
			var req Req
			if err := msg.Decode(&req); err != nil {
				return nil, err
			}
			return s.svc.Sum(ctx, &req)

		default:
			return nil, status.Error(codes.Unimplemented, "")
		}
	}, opts...)
	if err != nil {
		return err
	}

	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	<-sigchan

	return nil
}

func NewServer(tp transport.Transport, svc Service) natsrpc.Server {
	return &server{
		tp:  tp,
		svc: svc,
	}
}
