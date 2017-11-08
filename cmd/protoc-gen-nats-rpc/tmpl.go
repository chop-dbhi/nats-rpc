package main

const tmpl = `package {{ .Pkg }}

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
{{ range .Methods }}	{{ .Name }}(context.Context, *{{ .InputType | base }}) (*{{ .OutputType | base}}, error)
{{ end }}}

type Client interface {
{{ range .Methods }}	{{ .Name }}(context.Context, *{{ .InputType | base }}, ...transport.RequestOption) (*{{ .OutputType | base}}, error)
{{ end }}}

// client is an implementation of Client.
type client struct {
	tp transport.Transport
}
{{ range .Methods }}func (c *client) {{ .Name }}(ctx context.Context, req *{{ .InputType | base }}, opts ...transport.RequestOption) (*{{ .OutputType | base}}, error) {
	var rep {{ .OutputType | base }}
	
	_, err := c.tp.Request("{{ .Topic }}", req, &rep, opts...)
	if err != nil {
		return nil, err
	}

	return &rep, nil
}

{{ end }}// NewClient creates a new {{ .Name }} client.
func NewClient(tp transport.Transport) Client {
	return &client{tp}
}

type server struct {
	tp  transport.Transport
	svc {{ .Name }}
}

func (s *server) Serve(ctx context.Context, opts ...transport.SubscribeOption) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error
	_, err = s.tp.Subscribe("{{ .Subject }}.>", func(msg *transport.Message) (proto.Message, error) {
		switch msg.Subject { {{ range .Methods }}
		case "{{.Topic}}":
			var req {{ .InputType | base }}
			if err := msg.Decode(&req); err != nil {
				return nil, err
			}
			return s.svc.{{ .Name }}(ctx, &req)
		{{ end }}
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

func NewServer(tp transport.Transport, svc {{ .Name }}) natsrpc.Server {
	return &server{
		tp:  tp,
		svc: svc,
	}
}
`
