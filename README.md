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
and run this application:

1. [Requirements](docs/requirements.md)
1. [Set up service credentials](docs/set-up-service-credentials.md)
1. [Set up storage](docs/set-up-storage.md)
1. [Set up a Kubernetes cluster](docs/set-up-storage.md)
1. Set up continuous image build
1. Set up continuous deployment
1. Lock secrets down to services

Also if you're interested in developing this application yourself:

1. [Running services outside containers](#)
1. [Running locally on Minikube](#)

## Running locally without containers

For quick dev-test cycle, you might want to just run it directly on your dev
machine.

```sh
# make sure GOPATH is set and this repo is cloned to
# src/github.com/ahmetb/coffeelog. cd in to this directory.

export GOOGLE_APPLICATION_CREDENTIALS=<path-to-service-account-file>

# Start user service
go run ./userdirectory/*.go --addr=:8001 --google-project-id=<PROJECT> 

# Start coffee/activity service
go run ./coffeedirectory/*.go --addr=:8002 \
     --user-directory-addr=:8001 \
     --google-project-id=<PROJECT>

# Start web frontend
cd web # we need ./static directory to be present
go run *.go --addr=:8000 --user-directory-addr=:8001 \
    --coffee-directory-addr=:8002 \
    --google-oauth2-config=<path-to-file> \
    --google-project-id=<PROJECT>
```

## Running locally on Minikube

    minikube start

Build the docker image on minikube node:

    eval $(minikube docker-env)
    make

Deploy:

    kubectl apply -f misc/kube/

(Do not forget to change the gcr.io image name in deployment.yml above.)

Find out minikube IP from `minikube ip` and application port from `kubectl get
svc`. web-local service is configured to run on :32000. Head to http://ip:32000
to visit the application.

If you want to login to the app, create fake domain name in /etc/hosts, like
coffee.io and map the `minikube ip` to this hostname and update your OAuth2
configuration on Google API Manager to `http://coffee.io:32000/oauth2callback`.

## Running on Google Container Engine

#### 1. Create a cluster

Make sure you have created a GKE cluster and obtained credentials:

    gcloud container clusters create --zone us-central1-a coffee
    gcloud container clusters get-credentials --zone us-central1-a coffee 

#### 2. Automate docker image builds

You can use `make` to build the images and push to your gcr.io manually,
however, setting up automated continuous builds using [Google Cloud Container
Builder][cb] is a nicer solution:

[cb]: https://cloud.google.com/container-builder/

1. Go to Cloud Platform Console &rarr; Container Registry &rarr; Build Triggers
   &rarr; Add Trigger
1. Pick the GitHub repository (you can just fork this repo)
1. Select the "cloudbuild.yml" option and specify the file path as that.
1. Create the trigger (and trigger the first build manually)
1. See if the image build succeeds.

#### 3. Enable continous deployment

Update `misc/kube/deployment.yml` with your `gcr.io/<project-id>`. Then deploy
manually:

    kubectl apply -f misc/kube/

Or automate continuous deployment:

- [CircleCI](http://circleci.com) to deploy the new versions of `misc/kube/*`
  from the source code automatically. (See circle.yml for that.)
- [Google Container Builder][cb] can also run `kubectl apply`. However it is not
  very pretty today.(TODO: explain more)

#### 4. Try it

Find out the External IP address of the exposed service by using `kubectl get
service/web` and visit the application at `http://IP`.

## Planned Features

- [ ] Integrate Kubernetes RBAC to control access to secrets from pods
- [ ] use linkerd (or [Kubernetes TLS](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/)) to encrypt traffic       between microservices
- [ ] Helm to package and deploy the application manifests
  - [ ] App Registry to store Helm charts

**Disclaimer:** This is not an official Google product.
