package main

var fileTmpl = `// Generated by nats-rpc. DO NOT EDIT.
package {{ .Pkg }}

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/chop-dbhi/nats-rpc/transport"
	"github.com/golang/protobuf/proto"
)

var (
	traceIdKey = struct{}{}
)

type Client interface {
	{{ .Name }}
}

type client struct {
	tp transport.Transport
}
{{ range .Methods }}
func (c *client) {{ .Name }}(ctx context.Context, req *{{ .Request.Type }}) (*{{ .Response.Type }}, error) {
  var rep {{ .Response.Type }}

  _, err := c.tp.Request({{ .Topic }}, req, &rep)
  if err != nil {
    return nil, err
  }

  return &rep, nil
}
{{ end }}
func NewClient(tp transport.Transport) Client {
	return &client{tp}
}

func Serve(ctx context.Context, tp transport.Transport, svc Service) error {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
	}()

	var err error

	{{ range .Methods }}
	_, err = tp.Subscribe({{ .Topic }}, func(msg *transport.Message) (proto.Message, error) {
		ctx := context.WithValue(ctx, traceIdKey, msg.Id)

		var req {{ .Request.Type }}
		if err := msg.Decode(&req); err != nil {
			return nil, err
		}

		return svc.{{ .Name }}(ctx, &req)
	}, transport.SubscribeQueue({{ .ServiceGroup }}))
	if err != nil {
		return err
	}
	{{ end }}

	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	<-sigchan

	return nil
}
`

var cliTmpl = `// Generated by nats-rpc. DO NOT EDIT.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"

	"{{ .PkgPath }}"

	"github.com/chop-dbhi/nats-rpc/log"
	"github.com/chop-dbhi/nats-rpc/transport"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/nats-io/go-nats"

	"go.uber.org/zap"
)

const (
	clientType = "{{ .Pkg }}-cli"
)

var (
	buildVersion string

	traceIdKey = struct{}{}

	jsonMarshaler = &jsonpb.Marshaler{
		EmitDefaults: true,
	}

	jsonUnmarshaler = &jsonpb.Unmarshaler{}
)

func main() {
	var (
		natsAddr     string
		printVersion bool
	)

	flag.StringVar(&natsAddr, "nats.addr", "nats://127.0.0.1:4222", "NATS address.")
	flag.BoolVar(&printVersion, "version", false, "Print version.")

	flag.Parse()

	if printVersion {
		fmt.Fprintln(os.Stdout, buildVersion)
		return
	}

	// Get method.
	args := flag.Args()

	if len(args) == 0 {
		log.Fatalf("method name required")
	}

	meth := args[0]

	// Initialize base logger.
	logger, err := log.New()
	if err != nil {
		log.Fatal(err)
	}

	logger = logger.With(
		zap.String("client.type", clientType),
		zap.String("client.version", buildVersion),
	)

	// Initialize the transport layer.
	tp, err := transport.Connect(&nats.Options{
		Url: natsAddr,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Close()

	tp.SetLogger(logger)

	inp := "{}"
	if len(args) > 1 {
		inp = args[1]
	}

	inpr := bytes.NewBufferString(inp)

	client := {{ .Pkg }}.NewClient(tp)

	var rep proto.Message
	ctx := context.Background()

	switch meth { {{ range .Methods }}
	case "{{ .Name }}":
		var req {{ .Pkg }}.{{ .Request.Type }}
		if err := jsonUnmarshaler.Unmarshal(inpr, &req); err != nil {
			log.Fatalf("json: %s", err)
		}
		rep, err = client.{{ .Name }}(ctx, &req)
		{{ end }}

	default:
		log.Fatalf("unknown method %s", meth)
	}

	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}

	if err := jsonMarshaler.Marshal(os.Stdout, rep); err != nil {
		log.Fatalf("error encoding response: %s", err)
	}
	fmt.Fprint(os.Stdout, "\n")
}
`
