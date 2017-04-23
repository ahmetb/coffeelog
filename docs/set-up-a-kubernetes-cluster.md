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
