# Transport

This is a Go library that provides a thin, but opinionated abstraction over the [go-nats](https://github.com/nats-io/go-nats) API. The use case is for writing services that use NATS as a transport layer.

The NATS API expects a slice of bytes as the representation of a message. In general, it is often necessary to standardize on a serialization format to simplify designed and interacting with messages.

This library standardizes on [Protocol Buffers](https://developers.google.com/protocol-buffers/) as the message serialization format and it provides a few conveniences when working with Protobuf messages.

The two main value-add features this library provides are the `Transport` interface and implementation and the `Message` type.

The `Transport` interface describes a set of API methods that take Protobuf messages rather than byte slices. Being an interface, it enables implementing wrapper *middleware* to the API itself for the purpose of instrumentation. For example:

```go
// Declare a transport and initialize the core implementation using an
// existing NATS connection.
var tp transport.Transport
tp = transport.New(nc)

// Define a struct that satisfies the Transport interface and takes another
// Transport value to wrap.
type timerTransport struct {
  tp transport.Transport
}

func (t *timerTransport) Request(sub string, req proto.Message, rep proto.Message) (*transport.Message, error) {
  t0 := time.Now()
  msg, err := t.tp.Request(sub, req, rep)
  log.Printf(time.Now().Sub(t0))
  return msg, err
}

// Implement remaining methods..

// Wrap the base transport.
tp = &timerTransport{tp}

// Using this wrapped transport will now automatically log the duration of the call.
tp.Request(...)
```

In the `Request` method shown above, a `transport.Message` value is returned. `Message` is a Protobuf message which wraps all messages sent through the API. It annotates the message with additional metadata such as:

- `id` - a unique message ID.
- `timestamp` - timestamp in nanoseconds.
- `cause` - the causal upstream message.
- `subject` - the subject of the message.
- `reply` - the reply subject of a request message.
- `queue` - the queue that handled the message.
- `error` - a handling error if one occurred.

This provides additional metadata on the message which can be useful for logging or instrumentation.

## Quickstart

To initialize a new transport client, use either `transport.Connect` with [NATS options](https://godoc.org/github.com/nats-io/go-nats#Options) or `transport.New` with an existing `*nats.Conn` value.

```go
tp, err := transport.Connect(&nats.Options{
  Url: "nats://localhost:4222",
})
defer tp.Close()
```

Deferring the close ensures all subscriptions are stopped and the connection is closed.

There are two ways to publish messages. The first is `Publish`, the standard "fire and forget" broadcast to all subscribers of the message subject.

```go
val := pb.Value{ ... }
msg, err := tp.Publish("query.sink", &val)
```

The payload must implement [`proto.Message`](https://godoc.org/github.com/golang/protobuf/proto#Message), meaning that is must be a Protobuf message.

The second way to publish messages is `Request`, which waits for a reply so it can return response data to the client.

```go
// Request message to send.
req := pb.Request{ ... }

// The protobuf message to decode the reply into.
var rep pb.Reply

msg, err := tp.Request("query.execute", &req, &rep)
```

Here's how to subscribe to a subject using `Subscribe`.

```go
// Define the handler.
hdlr := func(msg *transport.Message) (proto.Message, error) {
  // Decode message payload into local request value.
  var req pb.Request
  if err := msg.Decode(&req); err != nil {
    return nil, err
  }
  // Do things..
  return &pb.Reply{ ... }, nil
}

// Subscribe the handler.
_, err := c.Subscribe("query.execute", hdlr)
```
