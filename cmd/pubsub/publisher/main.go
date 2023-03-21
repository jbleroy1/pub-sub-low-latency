package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"

	"cloud.google.com/go/pubsub"
	envconfig "github.com/sethvargo/go-envconfig"
)

var conf Config

func init() {

}

func setup() (*pubsub.Client, string) {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, conf.ProjectID)
	if err != nil {
		log.Fatalf("failed to create client: %s", err)
	}
	topicID := fmt.Sprintf("%s", conf.TopicPrefix)
	return client, topicID
}

func main() {
	ctx := context.Background()
	if err := envconfig.Process(ctx, &conf); err != nil {
		log.Fatalf("invalid env config: %s", err)
	}

	client, topicID := setup()
	defer client.Close()

	topic := client.Topic(topicID)

	ticker := time.NewTicker(time.Duration(1000/conf.Rps) * time.Millisecond)

	var publishedMsgs, numErrors uint64
	publishedMsgsChan := make(chan bool, 10)
	errorsChan := make(chan bool, 10)
	counterDone := make(chan bool)

	go counter(counterDone, publishedMsgsChan, errorsChan, &publishedMsgs, &numErrors)

	dispatcherDone := make(chan bool)
	go dispatchPublisher(ctx, topic, dispatcherDone, ticker, conf.MessageSize, publishedMsgsChan, errorsChan)

	time.Sleep(conf.Duration)
	ticker.Stop()
	counterDone <- true
	dispatcherDone <- true

	log.Printf("successfully published %d messages, failed %d ", publishedMsgs, numErrors)
}

func counter(done, publishedMsgsChan, errorsChan chan bool, publishedMsgs, numErrors *uint64) {
	for {
		select {
		case <-done:
			return
		case <-publishedMsgsChan:
			atomic.AddUint64(publishedMsgs, 1)
		case <-errorsChan:
			atomic.AddUint64(numErrors, 1)
		}
	}
}

func dispatchPublisher(ctx context.Context, topic *pubsub.Topic, done chan bool, ticker *time.Ticker, msgSize int, publishedMsgsChan, numErrsChan chan bool) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// If a tick comes in, publish a message.
			go publish(ctx, topic, msgSize, publishedMsgsChan, numErrsChan)
		}
	}
}

func publish(ctx context.Context, topic *pubsub.Topic, msgSize int, publishedMsgsChan, numErrsChan chan bool) {
	attrs := map[string]string{
		// The pt attribute is the publication time in nanoseconds since epoch.
		"pt": strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	data := bytes.Repeat([]byte("A"), int(math.Max(float64(msgSize-int(unsafe.Sizeof(attrs))-25), 1)))
	msg := pubsub.Message{
		Data:       data,
		Attributes: attrs,
	}

	result := topic.Publish(ctx, &msg)

	// The Get method blocks until a server-generated ID or
	// an error is returned for the published message.
	if _, err := result.Get(ctx); err != nil {
		log.Fatalf("invalid published message : %s", err)
		numErrsChan <- true
		return
	}

	publishedMsgsChan <- true
}

type Config struct {
	// PubSub
	ProjectID   string        `env:"projectID"`
	TopicPrefix string        `env:"topicPrefix"`
	Duration    time.Duration `env:"duration"`
	MessageSize int           `env:"messageSize"`
	Rps         int           `env:"RPS"`
}
