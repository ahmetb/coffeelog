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

and update the `misc/kube/configmap-google.yaml` to your Google Cloud project ID.

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

Make sure you have created a GKE cluster and obtained credentials:

    gcloud container clusters create --zone us-central1-a coffee
    gcloud container clusters get-credentials --zone us-central1-a coffee 

You can directly push these images to your gcr.io space and edit image names in
deployment.yml, however, setting up continuous builds using [Google Cloud
Container Builder][https://cloud.google.com/container-builder/] is a nicer
solution:

1. Go to Cloud Platform Console &rarr; Container Registry &rarr; Build Triggers
   &rarr; Add Trigger
1. Pick the GitHub repository (you can just fork this repo)
1. Select the "cloudbuild.yml" option and specify the file path as that.
1. Create the trigger (and trigger the first build manually)
1. See if the image build succeeds.

Deploy manually:

    kubectl apply -f misc/kube/

(or use circle.yml to set up a CircleCI build to deploy the new version from
source code automatically.)

Find out the External IP address of the exposed service by using
`kubectl get service/web` and visit the application at `http://IP`.
