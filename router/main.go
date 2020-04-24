// Sources for https://watermill.io/docs/getting-started/
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

var (
	// For this example, we're using just a simple logger implementation,
	// You probably want to ship your own implementation of `watermill.LoggerAdapter`.
	logger = watermill.NewStdLogger(false, false)
)

func main() {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

	// SignalsHandler will gracefully shutdown Router when SIGTERM is received.
	// You can also close the router by just calling `r.Close()`.
	router.AddPlugin(plugin.SignalsHandler)

	// Router level middleware are executed for every message sent to the router
	router.AddMiddleware(
		// CorrelationID will copy the correlation id from the incoming message's metadata to the produced messages
		middleware.CorrelationID,

		// The handler function is retried if it returns an error.
		// After MaxRetries, the message is Nacked and it's up to the PubSub to resend it.
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: time.Millisecond * 100,
			Logger:          logger,
		}.Middleware,

		// Recoverer handles panics from handlers.
		// In this case, it passes them as errors to the Retry middleware.
		middleware.Recoverer,
	)

	// For simplicity, we are using the gochannel Pub/Sub here,
	// You can replace it with any Pub/Sub implementation, it will work the same.
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)

	// AddHandler returns a handler which can be used to add handler level middleware
	handler := router.AddHandler(
		"struct_handler",          // handler name, must be unique
		"incoming_messages_topic", // topic from which we will read events
		pubSub,
		"outgoing_messages_topic", // topic to which we will publish events
		pubSub,
		(&structHandler{}).Handler,
	)

	// Handler level middleware is only executed for a specific handler
	// Such middleware can be added the same way the router level ones
	handler.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(message *message.Message) ([]*message.Message, error) {
			log.Println("^^^^^^^ executing handler specific middleware for ", message.UUID)
			return h(message)
		}
	})

	// just for debug, we are printing all messages received on `incoming_messages_topic`
	router.AddNoPublisherHandler(
		"print_incoming_messages",
		"incoming_messages_topic",
		pubSub,
		printMessages,
	)

	// just for debug, we are printing all events sent to `outgoing_messages_topic`
	router.AddNoPublisherHandler(
		"print_outgoing_messages",
		"outgoing_messages_topic",
		pubSub,
		printMessages,
	)

	// Producing some incoming messages in background
	go publishMessages(pubSub)

	// Now that all handlers are registered, we're running the Router.
	// Run is blocking while the router is running.
	ctx := context.Background()
	if err := router.Run(ctx); err != nil {
		panic(err)
	}

}

func publishMessages(publisher message.Publisher) {
	// for {
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, world!"))
		middleware.SetCorrelationID(watermill.NewUUID(), msg)

		fmt.Println()
		log.Printf("sending message %s, correlation id: %s\n", msg.UUID, middleware.MessageCorrelationID(msg))

		if err := publisher.Publish("incoming_messages_topic", msg); err != nil {
			panic(err)
		}

		// }
	}
}

func printMessages(msg *message.Message) error {
	fmt.Printf(
		"\n>> message printer\n> Received message: %s\n> %s\n> metadata: %v\n\n",
		msg.UUID, string(msg.Payload), msg.Metadata,
	)
	return nil
}

type structHandler struct {
	// we can add some dependencies here
	uuids []string
}

func (s *structHandler) Handler(msg *message.Message) ([]*message.Message, error) {
	s.uuids = append(s.uuids, msg.UUID)
	if len(s.uuids) < 10 {
		return nil, nil
	}
	log.Printf("received messages: %v\n", s.uuids)
	msg = message.NewMessage(watermill.NewUUID(), []byte(strings.Join(s.uuids, "\n")))
	s.uuids = make([]string, 0, 10)
	return message.Messages{msg}, nil
}
