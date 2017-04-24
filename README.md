# coffeelog

CoffeeLog is a multi tier web application where people can create an account
with their Google accounts, post pictures and other details of their coffee
experiences.

It is intended to be a demo application which is written to demonstrate and
test new DevOps technologies and features of Google Cloud Platform. **You can
deploy this application yourself and play with it.**

It uses:

- Go programming language
- gRPC
- Google Cloud Datastore
- Google Cloud Storage
- Kubernetes on Google Container Engine
- Stackdriver Logging
- Stackdriver Trace

## Setup

The following steps will walk you through on how to prepare requirements, deploy
and run this application.

> **Note:** If you see any issues with the steps below, please [open an
issue](https://github.com/ahmetb/coffeelog/issues/new).

1. [Requirements](docs/requirements.md)
1. [Set up service credentials](docs/set-up-service-credentials.md)
1. [Set up storage](docs/set-up-storage.md)
1. [Set up a Kubernetes cluster](docs/set-up-storage.md)
1. [Set up continuous image build](docs/set-up-image-build.md)
1. [Set up continuous deployment](docs/set-up-continuous-build.md)
1. [Try out the application!](docs/try-out.md)
1. Lock secrets down to services

Also if you're interested in developing this application yourself:

1. [Running services outside containers](docs/run-directly.md)
1. [Running locally on Minikube](docs/run-minikube.md)

-----

**Disclaimer:** This is not an official Google product.
