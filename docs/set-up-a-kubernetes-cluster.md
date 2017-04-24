# Set up a Kubernetes cluster

Coffeelog consists of several microservices, which are deployed
as [Docker](https://docker.com) containers.

To orchestrate deployment, lifecycle and replication of these services
on a pool of [compute instances](https://cloud.google.com/compute), we
use [Kubernetes](https://kubernetes.io).

## Create a Container Engine cluster

To create a Container Engine cluster named `coffee` with
3 nodes (and node auto-scaling enabled), run:

    gcloud container clusters create \
       --zone us-central1-a \
       --num-nodes 3 \
       --enable-autoscaling --min-nodes 1 --max-nodes 10 \
       coffee

Once it is succeeded, `kubectl` will be configured to use
the cluster. Run the following command to verify:

    kubectl get nodes

## Import secrets

Now import the two keys created earlier in the “Set up service credentials” step as [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

To save the service account, run:

    kubectl create secret generic google-service-account --from-file=app_default_credentials.json=<PATH_TO_FILE>

To save the OAuth2 key secret, run:

    kubectl create secret generic oauth2 --from-file=client-secret.json=<PATH_TO_FILE>

## Update configuration

The `misc/kube/configmap-google.yaml` will be deployed in the next steps. It
contains a [ConfigMap
resource](https://kubernetes.io/docs/tasks/configure-pod-container/configmap/)
which has account-specific values passed to the application.

Edit the following configuration keys:

- `project.id`: this is your Google Cloud project ID.
- `pics_bucket.name`: name of the storage bucket you created earlier in “Set up
  storage” step.

Then, commit and push changes (to your fork).

## Manual deployment

Open `misc/kube/deployment.yaml` and change the image names `gcr.io/PROJECT_ID`
to your actual project ID. Commit and push changes (to your fork). Then, deploy
everything using the following command:

    kubectl apply -f ./misc/kube

Ideally, you should set up automated [continuous image
builds](docs/set-up-image-build.md) and [continuous
deployments](docs/set-up-continuous-build.md).

## Try out manual deployment

Find out the load balancer public IP address of the web frontend:

    $ kubectl get service web

    NAME      CLUSTER-IP      EXTERNAL-IP       PORT(S)        AGE
    web       10.27.254.143   104.154.242.220   80:32428/TCP   55d

It can take a while for external IP to appear. Once it does, you can create a
hostname by appending `xip.io` (e.g. http://104.154.242.220.xip.io) and visit
the website to see if it works.

Using this hostname, you can go back to [API
Manager](https://console.cloud.google.com/apis/dashboard) and edit the callback
URL from `localhost` to the `IP.AD.DR.ESS.xip.io` format and log in to the
application!
