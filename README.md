# coffeelog

CoffeeLog is a multi tier web application where people can create an account
with their Google accounts, post pictures and other details of their coffee
experiences.

It is intended to be a demo application which is written to demonstrate and
test new DevOps technologies and features of Google Cloud Platform. It uses:

- Go programming language
- gRPC
- Google Cloud Datastore
- Google Cloud Storage
- Kubernetes on Google Container Engine
- Stackdriver Logging

**Disclaimer:** This is not an official Google product.

## Required Configuration

1. a Google Cloud Platform project ID
1. Create a Service Account with following roles and download the JSON key:
  - Datastore User
  - Storage Admin
1. Create an OAuth2 client and download the JSON key.
  - Google Cloud Console &rarr; API Manager &rarr; Credentials &rarr; Create &rarr; OAuth client ID
  - You can specify callback uri as `http://localhost/oauth2callback` and change later.

## Setup

Create Datastore indexes required:

    gcloud datastore create-indexes misc/index.yaml

If you are going to deploy on Kubernetes, add keys as secrets:

    kubectl create secret generic google-service-account --from-file=app_default_credentials.json=<path-to-file-on-disk>
    kubectl create secret generic oauth2 --from-file=client-secret.json=<path-to-file-on-disk>

## Running locally on Minikube

    minikube start

Build the docker image on minikube node:

    eval $(minikube docker-env)
    docker build -t gcr.io/ahmetb-starter/monoimage:latest .

Deploy:

    kubectl apply -f misc/service.yml
    kubectl apply -f misc/deployment.yml

Find out minikube IP from `minikube ip` and application port from `kubectl get svc`.
web-local service is configured to run on :32000. Head to http://ip:32000 to visit
the application.

If you want to login to the app, create fake domain name in /etc/hosts, like coffee.io
and map the `minikube ip` to this hostname and update your OAuth2 configuration on
Google API Manager to `http://coffee.io:32000/oauth2callback`.

## Running on Google Container Engine

TODO explain local building instructions

TODO explain setting up cloud build


Deploy:

    kubectl apply -f misc/service.yml
    kubectl apply -f misc/deployment.yml
