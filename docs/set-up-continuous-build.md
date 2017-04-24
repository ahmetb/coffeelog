# Set up continuous build

Deploying this application is as simple as deploying all the manifests under
[`misc/kube`](/misc/kube) directory using `kubectl` (requires changing the
hard-coded image names).

However ideally you want a new build to go live automatically when you push
a new commit to your GitHub repository. For that, we can use a third-party
Continuous Integration (CI) solution such as [CircleCI](http://circleci.com).

## Setting up CircleCI

[Create a new Service
Account](https://console.cloud.google.com/iam-admin/serviceaccounts/) on Cloud
Console. Give it a name (e.g. `cirleci-deployment`) and give it the role
“Container Engine Developer”. Click "furnish a new private key" and choose JSON
type. Download this file.

1. Sign up for an account on [CircleCI](http://circleci.com) with your GitHub
account.

1. Enable the builds on the `coffeelog` repository. This will use the
[`circle.yml`](/circle.yml).

1. Go to settings of the repository on Circle CI and click "Environment Variables"
   &rarr; "Add Variable".

1. Specify key as `GCLOUD_SERVICE_KEY` and paste the base64 encoding of the
   JSON key file as the value. You can use this to encode:

       cat file.json | base64 -w0

1. Go to the Builds tab, and start a new build by clicking "Rebuild" on one of
   the previous builds.

1. Watch the logs, see if the build succeeds.


`circle.yml` file basically uses the Service Account key to authenticate
to your Container Engine cluster, then modifies the `:latest` tag in the
`misc/kube/*.yaml` files with the Git commit ID and uses `kubectl apply -f`
command do deploy everything.

Since we use [Deployment
contoller](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
on Kubernetes, the deployment operation will be a rollingu update and if
anything fails (e.g. image build that happens asynchronously can fail), existing
instances will keep running.
