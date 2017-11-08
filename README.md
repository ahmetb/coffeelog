# coffeelog

Coffeelog is a multi-tier web application where coffee enthusiasts can log in
with their Google accounts and share their coffee experiences with other people.

This is intended to be a sample cloud-native application to showcase the best
practices in application deployment, products of [Google Cloud](https://cloud.google.com)
and test new features.

Coffeelog is written in [Go](https://golang.org), uses [gRPC](https://grpc.io)
for communication between microservices. It runs on [Google
Cloud](https://cloud.google.com) and uses Cloud Datastore, Cloud Storage,
[Google Container Engine](https://cloud.google.com/container-engine/), [Cloud
Container Builder](https://cloud.google.com/container-builder/), [Stackdriver
Logging](https://cloud.google.com/logging/) and [Stackdriver
Trace](https://cloud.google.com/trace/).

## Setup

The following steps will walk you through on how to prepare requirements, deploy
and run this application.

> **Note:** If you see any issues with the steps below, please [open an
issue](https://github.com/ahmetb/coffeelog/issues/new).

1. [Requirements](docs/requirements.md)
1. [Set up service credentials](docs/set-up-service-credentials.md)
1. [Set up storage](docs/set-up-storage.md)
1. [Set up a Kubernetes cluster on Google Container Engine](docs/set-up-a-kubernetes-cluster.md)
1. [Set up continuous image build on Container Builder](docs/set-up-image-build.md)
1. [Set up continuous deployment on Container Builder](docs/set-up-continuous-build.md)
1. [Try out the application!](docs/try-out.md)

**Monitoring:**

1. :soon: Set up distributed tracing with Stackdriver Trace
1. :soon: Browse application logs with Stackdriver Logging
1. :soon: Set up alerting with Stackdriver Monitoring

**Advanced topics:**

1. :soon: Set up a domain name
1. :soon: Set up TLS with Let’s Encrypt
1. :soon: Limit access to secrets with Kubernetes RBAC and Service accounts
1. :soon: Set up TLS communication between microservices with linkerd
1. :soon: [Set up Network Policy to restrict communication between services](docs/network-policy.md)

**Local development** (if you are interested in developing this app further)

1. [Running services outside containers](docs/run-directly.md)
1. [Running locally on Minikube](docs/run-minikube.md)

-----

**Disclaimer:** This is not an official Google product.
