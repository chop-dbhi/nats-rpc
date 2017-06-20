package transport

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nuid"
	"go.uber.org/zap"
)

var (
	DefaultRequestTimeout = 2 * time.Second
)

// PublishOptions are options for a publication.
type PublishOptions struct {
	Cause string
}

type PublishOption func(*PublishOptions)

// PublishCause sets the cause of the publication.
func PublishCause(s string) PublishOption {
	return func(o *PublishOptions) {
		o.Cause = s
	}
}

// RequestOptions are options for a publication.
type RequestOptions struct {
	Cause   string
	Timeout time.Duration
}

type RequestOption func(*RequestOptions)

// RequestTimeout sets a request timeout duration.
func RequestTimeout(t time.Duration) RequestOption {
	return func(o *RequestOptions) {
		o.Timeout = t
	}
}

// RequestCause sets the cause of the request.
func RequestCause(s string) RequestOption {
	return func(o *RequestOptions) {
		o.Cause = s
	}
}

// SubscribeOptions are options for a subscriber.
type SubscribeOptions struct {
	Queue string
}

type SubscribeOption func(*SubscribeOptions)

// SubscribeQueue specifies the queue name of the subscriber.
func SubscribeQueue(q string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Queue = q
	}
}

// Decode decodes the message payload into a proto message.
func (m *Message) Decode(pb proto.Message) error {
	return proto.Unmarshal(m.Payload, pb)
}

// Handler is the handler used by a subscriber. The return value may be nil if
// no output is yielded. If this is a request, the reply will be sent automatically
// with the reply value or an error if one occurred. If a reply is not expected
// and error occurs, it will be logged.
type Handler func(msg *Message) (proto.Message, error)

// Transport describes the interface
type Transport interface {
	// Publish publishes a message asynchronously to the specified subject.
	// The wrapped message is returned or an error. The error would only be due to
	// a connection issue, but does not reflect any consumer error.
	Publish(sub string, msg proto.Message, opts ...PublishOption) (*Message, error)

	// Request publishes a message synchronously and waits for a response that
	// is decoded into the Protobuf message supplied. The wrapped message is
	// returned or an error. The error could be a connection error, timeout,
	// or a consumer error.
	Request(sub string, req proto.Message, rep proto.Message, opts ...RequestOption) (*Message, error)

	// Subscribe creates a subscription to a subject.
	Subscribe(sub string, hdl Handler, opts ...SubscribeOption) (*nats.Subscription, error)

	// Conn returns the underlying NATS connection.
	Conn() *nats.Conn

	// Close closes the transport connection and unsubscribes all subscribers.
	Close()

	// Set the logger.
	SetLogger(*zap.Logger)
}

// Connect is a convenience function establishing a connection with
// NATS and returning a transport.
func Connect(opts *nats.Options) (Transport, error) {
	conn, err := opts.Connect()
	if err != nil {
		return nil, err
	}

	logger, _ := zap.NewProduction()
	return &transport{
		logger: logger,
		conn:   conn,
	}, nil
}

// New returns a transport using an existing NATS connection.
func New(conn *nats.Conn) Transport {
	logger, _ := zap.NewProduction()

	return &transport{
		logger: logger,
		conn:   conn,
	}
}

type transport struct {
	logger *zap.Logger
	conn   *nats.Conn
	subs   []*nats.Subscription
	mux    sync.Mutex
}

func (c *transport) SetLogger(l *zap.Logger) {
	c.logger = l
}

func (c *transport) Conn() *nats.Conn {
	return c.conn
}

func (c *transport) Close() {
	for _, sub := range c.subs {
		sub.Unsubscribe()
	}
	c.conn.Close()
}

func (c *transport) wrap(payload proto.Message) (*Message, error) {
	var (
		pb  []byte
		err error
	)

	if payload != nil {
		pb, err = proto.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	id := nuid.Next()
	ts := time.Now().UnixNano()

	msg := Message{
		Id:        id,
		Timestamp: uint64(ts),
		Payload:   pb,
	}

	return &msg, nil
}

func (c *transport) unwrap(nmsg *nats.Msg) (*Message, error) {
	var msg Message

	if err := proto.Unmarshal(nmsg.Data, &msg); err != nil {
		return nil, err
	}

	msg.Subject = nmsg.Subject
	msg.Reply = nmsg.Reply
	msg.Queue = nmsg.Sub.Queue

	return &msg, nil
}

func (c *transport) Publish(sub string, msg proto.Message, opts ...PublishOption) (*Message, error) {
	pubOpts := &PublishOptions{}

	// Apply options.
	for _, opt := range opts {
		opt(pubOpts)
	}

	m, err := c.wrap(msg)
	if err != nil {
		return nil, err
	}

	m.Subject = sub
	m.Cause = pubOpts.Cause

	mb, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}

	if err := c.conn.Publish(sub, mb); err != nil {
		return nil, err
	}

	return m, nil
}

func (c *transport) Request(sub string, req proto.Message, rep proto.Message, opts ...RequestOption) (*Message, error) {
	reqOpts := &RequestOptions{
		Timeout: DefaultRequestTimeout,
	}

	// Apply options.
	for _, opt := range opts {
		opt(reqOpts)
	}

	m, err := c.wrap(req)
	if err != nil {
		return nil, err
	}

	m.Subject = sub
	m.Cause = reqOpts.Cause

	mb, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}

	nm, err := c.conn.Request(sub, mb, reqOpts.Timeout)
	if err != nil {
		return nil, err
	}

	m, err = c.unwrap(nm)
	if err != nil {
		return nil, err
	}

	// Error occurred in the handler.
	if m.Error != "" {
		return nil, errors.New(m.Error)
	}

	if rep != nil {
		if err := proto.Unmarshal(m.Payload, rep); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// Subscribe creates a subscription to a subject.
func (c *transport) Subscribe(sub string, hdlr Handler, opts ...SubscribeOption) (*nats.Subscription, error) {
	subOpts := &SubscribeOptions{}

	// Apply options.
	for _, opt := range opts {
		opt(subOpts)
	}

	// Replies to the recipient with an error if applicable.
	replyWithError := func(logger *zap.Logger, msg *Message, srcErr error) {
		rmsg, err := c.wrap(nil)
		// If an error occurs, this is a bug since this only relies on the local
		// Message protobuf definition.
		if err != nil {
			logger.Error("failed to create transport message",
				zap.Error(err),
			)
			return
		}

		rmsg.Cause = msg.Id
		rmsg.Subject = msg.Reply
		rmsg.Error = srcErr.Error()

		mb, err := proto.Marshal(rmsg)
		if err != nil {
			logger.Error("failed to marshal transport message",
				zap.Error(err),
			)
			return
		}

		if err := c.conn.Publish(msg.Reply, mb); err != nil {
			logger.Error("failed to publish nats message",
				zap.Error(err),
			)
		}
	}

	// NATS message handler.
	natsHandler := func(nmsg *nats.Msg) {
		// Copy logger for this request.
		logger := c.logger.With(
			zap.String("msg.subject", nmsg.Subject),
			zap.String("msg.reply", nmsg.Reply),
		)

		msg, err := c.unwrap(nmsg)

		// Failed unwrap which means the message is likely in the wrong format.
		// A reply is ignored if this occurs since if the sent message was invalid
		// it is unlikely the requester will be able to parse the message in the
		// same format. Instead we log this case.
		if err != nil {
			logger.Error("failed to decode nats message")
			return
		}

		// Add more context.
		logger = c.logger.With(
			zap.String("trace.id", msg.Id),
			zap.String("msg.id", msg.Id),
			zap.String("msg.cause", msg.Cause),
		)

		// In case the handler panics, catch and log.
		defer func() {
			if rec := recover(); rec != nil {
				err := fmt.Errorf("recovered subscription handler panic:\n%s", rec)

				if msg.Reply == "" {
					logger.Error("subscription handler panic",
						zap.Error(err),
					)
					return
				}

				replyWithError(logger, msg, err)
			}
		}()

		// Pass to handler.
		resp, err := hdlr(msg)

		// Log error only if no reply.
		if msg.Reply == "" {
			if err != nil {
				logger.Error("subscription handler error",
					zap.Error(err),
				)
			}
			return
		}

		// Error occured while handling message. We assume this is an error in the
		// business logic and will reply with the error. The handler itself should
		// have logged the error if it occurred since it can provide context.
		if err != nil {
			replyWithError(logger, msg, err)
			return
		}

		// This will only fail if the response itself cannot be marshaled which
		// means the handler is likely at fault. The error should be logged
		// and a reply with a faulty handler can be returned.
		rmsg, err := c.wrap(resp)
		if err != nil {
			logger.Error("failed to marshal response message",
				zap.Error(err),
			)

			replyWithError(logger, msg, err)
			return
		}

		rmsg.Cause = msg.Id
		rmsg.Subject = msg.Reply

		// Bug.
		mb, err := proto.Marshal(rmsg)
		if err != nil {
			logger.Error("failed to marshal transport message",
				zap.Error(err),
			)
			return
		}

		if err := c.conn.Publish(msg.Reply, mb); err != nil {
			logger.Error("failed to publish nats message",
				zap.Error(err),
			)
		}
	}

	// Queue-based subscriber.
	if subOpts.Queue != "" {
		s, err := c.conn.QueueSubscribe(sub, subOpts.Queue, natsHandler)
		if err != nil {
			return nil, err
		}
		c.mux.Lock()
		c.subs = append(c.subs, s)
		c.mux.Unlock()
		return s, nil
	}

	// Standalone subscriber.
	s, err := c.conn.Subscribe(sub, natsHandler)
	if err != nil {
		return nil, err
	}
	c.mux.Lock()
	c.subs = append(c.subs, s)
	c.mux.Unlock()
	return s, nil
}
