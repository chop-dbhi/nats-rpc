package main

import (
	"context"
	"flag"
	"os"

	"github.com/chop-dbhi/nats-rpc/example"

	"github.com/chop-dbhi/nats-rpc/log"
	"github.com/chop-dbhi/nats-rpc/transport"

	"github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

func main() {
	var natsAddr string

	flag.StringVar(&natsAddr, "nats.addr", "nats://localhost:4222", "Address to NATS broker.")
	flag.Parse()

	// Initia,ize base logger.
	logger, err := log.New()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the transport layer.
	tp, err := transport.Connect(&nats.Options{
		Url: natsAddr,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Close()

	tp.SetLogger(logger)

	// Initialize the service.
	svc := example.NewService()

	ctx := context.Background()

	// Initialize a server and serve the service.
	srv := example.NewServer(tp, svc)
	if err := srv.Serve(ctx); err != nil {
		logger.Error("serve error", zap.Error(err))
		os.Exit(1)
	}
}
