# Running on local Minikube cluster

Download [minikube](https://github.com/kubernetes/minikube/) and create a local
cluster.

Build the docker image directly on the Docker engine of the minikube node:

    eval $(minikube docker-env)

    PROJECT=<your-project-id> make docker-images

Make sure you updated `image:` in `misc/kube/deployment.yaml` to use your
project ID instead of the `gcr.io/ahmet-...` format.

Deploy everything:

    kubectl apply -f ./misc/kube/

Find out minikube IP from `minikube ip` and application port from `kubectl get
svc`. web-local service is configured to run on :32000. Head to http://ip:32000
to visit the application.

If you want to login to the app, create fake domain name in /etc/hosts, like
coffee.io and map the `minikube ip` to this hostname and update your OAuth2
configuration on [API Manager console](https://console.cloud.google.com/apis/dashboard)
to `http://coffee.io:32000/oauth2callback`.
