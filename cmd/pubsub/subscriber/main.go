package main

import (
	"context"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	envconfig "github.com/sethvargo/go-envconfig"
)

var conf Config

func main() {
	ctx := context.Background()
	if err := envconfig.Process(ctx, &conf); err != nil {
		log.Fatalf("invalid env config: %s", err)
	}
	if conf.SubscriptionID == "" {
		log.Fatalf("subscriptionID must be set")
	}
	client := setup()
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, conf.TestDuration)
	defer cancel()

	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "pubsub",
		Name:      "latency",
		Help:      "Cloud Pub/Sub latency in milliseconds",
		Objectives: map[float64]float64{
			0.5:   0.05,
			0.9:   0.01,
			0.95:  0.005,
			0.99:  0.001,
			0.999: 0.0001,
		},
	})
	prometheus.MustRegister(summary)

	// sub.ReceiveSettings defaults to pubsub.DefaultReceiveSettings
	sub := client.SubscriptionInProject(conf.SubscriptionID, conf.ProjectID)
	burnInStart := time.Now().Unix()

	go receive(sub, ctx, burnInStart, summary)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func receive(sub *pubsub.Subscription, ctx context.Context, burnInStart int64, summary prometheus.Summary) {
	// sub.Receive calls the function passed into it concurrently from multiple goroutines.
	// So it's already async and parallelized.
	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		receiptTime := time.Now().UnixNano()
		ptAttr := msg.Attributes["pt"]

		publishTime, err := strconv.ParseInt(ptAttr, 10, 64)
		if err != nil {
			log.Printf("error converting %s to int64", ptAttr)
			msg.Ack()
			return
		}

		latencyMillis := float64(receiptTime-publishTime) / math.Pow(10, 6)
		burnInDurationLeft := int64(conf.BurnInDuration.Seconds()) - (time.Now().Unix() - burnInStart)
		if burnInDurationLeft < 0 {
			summary.Observe(latencyMillis)
		} else {
			log.Printf("burning in for %d more seconds", burnInDurationLeft)
		}
		// Uncomment next line for debugging
		//log.Printf("got message at %d, latency %.0f ms", receiptTime, latencyMillis)
		msg.Ack()
	})
	if err != nil {
		log.Printf("error pulling messages from subscription %s: %s", conf.SubscriptionID, err.Error())
	}
}

func setup() *pubsub.Client {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, conf.ProjectID)
	if err != nil {
		log.Fatalf("failed to create client: %s", err)
	}

	return client
}

type Config struct {
	// PubSub
	ProjectID      string        `env:"projectID"`
	SubscriptionID string        `env:"subscriptionID"`
	TestDuration   time.Duration `env:"testDuration"`
	BurnInDuration time.Duration `env:"burnInDuration"`
}
