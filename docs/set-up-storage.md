# Set up Cloud Datastore

Coffeelog uses [Google Cloud Datastore](https://cloud.google.com/datastore) as its
NoSQL database.

Visit [Google Cloud Console &rarr; Datastore](https://console.cloud.google.com/datastore/)
and pick a region for the Datastore.

Then, create the Cloud Datastore indexes required by
the application:

    gcloud datastore create-indexes misc/index.yaml

# Set up Cloud Storage

Coffeelog uses [Google Cloud Storage](https://cloud.google.com/storage) to store pictures
uploaded by users and to serve them on the website.

Create a new storage bucket with the NAME of your choosing and make it publicly readable:

    gsutil mb gs://NAME
    gsutil defacl ch -u AllUsers:R gs://NAME

You use the name of this bucket in one of the next steps.
