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

type {{ .Name }} interface {
{{ range .Methods }}	{{ .Name }}(context.Context, *{{ .InputType | base }}) (*{{ .OutputType | base}}, error)
{{ end }}}

type {{ .Name }}Client interface {
{{ range .Methods }}	{{ .Name }}(context.Context, *{{ .InputType | base }}, ...transport.RequestOption) (*{{ .OutputType | base}}, error)
{{ end }}}

// {{ .Name | lower }}Client an implementation of {{ .Name }} client.
type {{ .Name | lower }}Client struct {
	tp transport.Transport
}
{{ $Service := .Name }}
{{ range .Methods }}func (c *{{ $Service | lower}}Client) {{ .Name }}(ctx context.Context, req *{{ .InputType | base }}, opts ...transport.RequestOption) (*{{ .OutputType | base}}, error) {
	var rep {{ .OutputType | base }}
	
	_, err := c.tp.Request("{{ .Topic }}", req, &rep, opts...)
	if err != nil {
		return nil, err
	}

	return &rep, nil
}

{{ end }}// New{{ .Name }}Client creates a new {{ .Name }} client.
func New{{ .Name }}Client(tp transport.Transport) {{ .Name }}Client {
	return &{{ .Name | lower }}Client{tp}
}

type {{ .Name }}Server struct {
	tp  transport.Transport
	svc {{ .Name }}
}

func New{{ .Name }}Server(tp transport.Transport, svc {{ .Name }}) *{{ .Name }}Server {
	return &{{ .Name }}Server{
		tp:  tp,
		svc: svc,
	}
}

func (s *{{ .Name }}Server) Serve(ctx context.Context, opts ...transport.SubscribeOption) error {
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
`
