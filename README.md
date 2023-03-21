# Message-latency

Basic app to play with the pubsub and how to deliver low latency messages. This app is build on top of the solution provided by [@davidxia](https://github.com/davidxia/cloud-message-latency)

# Build

Build the docker image and push it on your Docker registry
```
docker build -t pubsub-publisher .
docker build -t pubsub-subscriber -f Dockerfile-sub .
```

# Deploy


Change the different values in the k8s/deployment-pub.yaml and k8s/deployment-sub.yaml files then simply apply.

# Prometheus

A PodMonitoring is defined and scrap metrics from subscriber deployment. Then you can observe the metrics in Cloud Monitoring with the Prometheus target

# Links

* [pubsub](https://cloud.google.com/pubsub/)
* [pubsub-architecture](https://cloud.google.com/pubsub/architecture#basic_architecture)
* [gcloud-install](https://cloud.google.com/sdk/docs/install)
* [go-install](https://golang.org/doc/install)
