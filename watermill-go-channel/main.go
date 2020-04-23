// Sources for https://watermill.io/docs/getting-started/
package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// TOPIC 定义了双方的话题
const TOPIC = "example.topic"

func main() {
	pubsub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer:            10,
			Persistent:                     true,
			BlockPublishUntilSubscriberAck: true,
		},
		watermill.NewStdLogger(false, false),
	)

	go sub1(1, pubsub)
	go sub2(2, pubsub)

	publishMessages(pubsub)

	// pubsub.Close()
}

func publishMessages(publisher message.Publisher) {
	// for i := 0; i < 10; i++ {
	i := 0
	for {
		// time.Sleep(time.Second * 1)
		i++
		id := strconv.Itoa(i)
		msg := message.NewMessage(id, []byte(id+", world!"))
		log.Printf("\tsending message\t: %s\n", string(msg.Payload))
		if err := publisher.Publish(TOPIC, msg); err != nil {
			panic(err)
		}
	}
}

func sub1(id int, sub message.Subscriber) {
	time.Sleep(time.Second * 20)
	messages, err := sub.Subscribe(context.Background(), TOPIC)
	if err != nil {
		panic(err)
	}
	for msg := range messages {
		log.Printf("%d,received message\t: %s\n", id, string(msg.Payload))
		// we need to Acknowledge that we received and processed the message,
		// otherwise, it will be resent over and over again.
		msg.Ack()
		os.Exit(1)
	}
}

func sub2(id int, sub message.Subscriber) {
	time.Sleep(time.Second * 10)
	messages, err := sub.Subscribe(context.Background(), TOPIC)
	if err != nil {
		panic(err)
	}
	for msg := range messages {
		log.Printf("%d,received message\t: %s\n", id, string(msg.Payload))
		// we need to Acknowledge that we received and processed the message,
		// otherwise, it will be resent over and over again.
		msg.Ack()
	}
}
