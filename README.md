# coffeelog

## Running on Minikube

```sh
# build the docker image to the local instance
minikube start
eval $(minikube docker-env)
docker build -t docker build -t gcr.io/ahmetb-starter/monoimage:latest .

# deploy the service
kubectl apply -f misc/service.yml
kubectl apply -f misc/deployment.yml
```
