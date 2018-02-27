# Set up continuous build

Deploying this application is as simple as deploying all the manifests under
[`misc/kube`](/misc/kube) directory using `kubectl` (requires changing the
hard-coded image names).

## Setting up on Container Builder

You can use [Google Cloud Container
Builder](https://cloud.google.com/container-builder/) to automate deployments
onto Google Kubernetes Engine.

### Give Container Builder Access to GKE

By default, Container Builder’s service account does not have permissions to
access Kubernetes Engine, so you have to make an one-time IAM [role
assignment](https://cloud.google.com/container-builder/docs/how-to/service-account-permissions)
manually.

This account is named `[PROJECT_ID]@cloudbuild.gserviceaccount.com` where the
`PROJECT_ID` is your **numeric** project ID. 

You can do this role assignment by going to the Cloud Platform Console &rarr;
IAM/Admin &rarr; IAM &rarr; Choose the service account with `cloudbuild` and
update its roles on the dropdown to add "Kubernetes Engine Developer" role.

**Or you can use `gcloud`**: First, find out your numeric project ID and
construct the cloudbuild account ID:

    PROJECT_NUM="$(gcloud projects list --filter='name = ahmetb-starter' --format='get(projectNumber)')"

Then assign this account the "Kubernetes Engine Developer" role:

    gcloud projects add-iam-policy-binding $PROJECT_NUM \
        --member="serviceAccount:$PROJECT_NUM@cloudbuild.gserviceaccount.com" \
        --role=roles/container.developer

### Create a build trigger

This will run the continuous deployment steps in `clouddeploy.yaml`
every time you push a commit to the `master` branch:

1. Fork this repository on GitHub
1. Go to Cloud Platform Console &rarr; Container Registry &rarr; [Build Triggers](https://console.cloud.google.com/gcr/triggers)
   &rarr; Add Trigger
1. Pick the coffeelog repository (that is, your fork of my repo) and Continue
1. Give it the name "Continuous depoyment".
1. Select the cloudbuild.yaml option and specify the file path as `clouddeploy.yaml`.
1. Specify the "Branch name" as `master`.
1. Create the trigger.
1. Trigger the first build manually.
1. See the logs if the deployment succeeds.

Note that this will run in parallel with the image build when you push a commit.
Therefore, when the manifests with the newer versions are applied to the
cluster, Deployment rollout will temporarily fail as it cannot find the image as
it is not pushed yet. It will retry and succeed, in the meanwhile some pods in
the Deployment will be unavailable but the services should continue to serve as
they have multiple replicas.
