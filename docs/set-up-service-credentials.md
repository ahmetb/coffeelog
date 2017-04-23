# Set up service credentials

## Create a [Service Account](https://cloud.google.com/compute/docs/access/service-accounts)

This gives the microservices access to use the Google Cloud APIs.

- Visit [Service Accounts] on Cloud Console(https://console.cloud.google.com/iam-admin/serviceaccounts)
- Click "Create Service Account"
- Give it a name, and choose roles:
  - [x] Datastore User
  - [x] Storage Admin
- Click "Furnish new private key" with type=JSON.
- Click "Create" and save the private key file.

You will use this file in "Set up a Kubernetes cluster" section later.

## Create an OAuth2 client

This is used for users with Google accounts to authenticate to the app.

- Visit [API Manager](https://console.cloud.google.com/apis/dashboard) on Cloud
  Console.
- Click "Credentials" &rarr; "Create credentials" &rarr; choose "OAuth client
  ID"
- Choose "Web application" on the next screen.
- Give it a name, and specify the callback URI as
  `http://localhost/oauth2callback`. You will change it once you have a domain
  name.

You will use this file in "Set up a Kubernetes cluster" section later.

