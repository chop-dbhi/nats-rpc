package transport

import (
	"log"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nuid"
)

func newTransport(t testing.TB) Transport {
	natsAddr := os.Getenv("NATS_ADDR")

	if natsAddr == "" {
		t.Fatal("no nats address provided")
	}

	tp, err := Connect(&nats.Options{
		Url: natsAddr,
	})

	if err != nil {
		t.Fatal(err)
	}

	return tp
}

func TestPublish(t *testing.T) {
	tp := newTransport(t)
	defer tp.Close()

	// Publish a message.
	msg, err := tp.Publish("_transport", nil, PublishCause("foobar"))
	if err != nil {
		t.Fatal(err)
	}

	if msg.Id == "" {
		t.Errorf("id not set")
	}

	if msg.Timestamp == 0 {
		t.Errorf("timestamp not set")
	}

	if msg.Subject != "_transport" {
		t.Errorf("wrong subject: %s", msg.Subject)
	}

	if len(msg.Payload) != 0 {
		t.Errorf("expected empty payload, got %d bytes", len(msg.Payload))
	}

	if msg.Cause != "foobar" {
		t.Errorf("expected foobar, got %s", msg.Cause)
	}

	if msg.Error != "" {
		t.Errorf("error set")
	}

	if msg.Queue != "" {
		t.Errorf("queue set")
	}

	if msg.Reply != "" {
		t.Errorf("reply set")
	}
}

func TestSubscribe(t *testing.T) {
	tp := newTransport(t)
	defer tp.Close()

	var (
		msg *Message
		err error
	)

	hdlr := func(cmsg *Message) (proto.Message, error) {
		if msg.Id != cmsg.Id {
			t.Errorf("wrong id")
		}
		return nil, nil
	}

	// Subscribe.
	_, err = tp.Subscribe("_transport", hdlr, SubscribeQueue("_queue"))
	if err != nil {
		t.Fatal(err)
	}

	msg, err = tp.Publish("_transport", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRequest(t *testing.T) {
	tp := newTransport(t)
	defer tp.Close()

	exp := &Message{
		Id: nuid.Next(),
	}

	// No-op reply.
	hdlr := func(cmsg *Message) (proto.Message, error) {
		if cmsg.Cause != "foobar" {
			t.Errorf("expected foobar, got %s", cmsg.Cause)
		}

		return exp, nil
	}

	// Subscribe.
	_, err := tp.Subscribe("_transport", hdlr, SubscribeQueue("_queue"))
	if err != nil {
		t.Fatal(err)
	}

	// Send request.
	var rep Message
	_, err = tp.Request("_transport", nil, &rep, RequestCause("foobar"))
	if err != nil {
		t.Fatal(err)
	}

	if rep.Id != exp.Id {
		t.Error("reply ids differ")
	}
}

func TestHandlerPanic(t *testing.T) {
	tp := newTransport(t)
	defer tp.Close()

	hdlr := func(cmsg *Message) (proto.Message, error) {
		var i *int
		log.Println(*i)
		return nil, nil
	}

	_, err := tp.Subscribe("_transport", hdlr)
	if err != nil {
		t.Fatal(err)
	}

	var rep Message
	_, err = tp.Request("_transport", nil, &rep)
	if err == nil {
		t.Errorf("expected error")
	}
}
