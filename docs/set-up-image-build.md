# Set up image build

You have two options to build Docker container images:

## 1. Manual image builds (not preferred)

You can run the following command to build the container images:

    PROJECT=<your-id> make docker-images

This will create 3 container images. Verify by running:

```sh
$ docker images
REPOSITORY                                  TAG        SIZE
gcr.io/foo/userdirectory                    latest     13.8 MB
gcr.io/foo/coffeedirectory                  latest     14.5 MB
gcr.io/foo/web                              latest     13.5 MB
```
You can then use the following command to push images to 
[Google Container Registry](https://cloud.google.com/container-registry/):

    gcloud docker -- push <image-name>

However, this method is not recommended as it is not automated and it does not
tag images with Git commit IDs to identify them (always tags as `:latest`).

## 2. Automated image builds (preferred)

You can up automate continuous builds using the [Google Cloud Container
Builder](https://cloud.google.com/container-builder/)

1. Fork this repository on GitHub
1. Go to Cloud Platform Console &rarr; Container Registry &rarr; [Build Triggers](https://console.cloud.google.com/gcr/triggers)
   &rarr; Add Trigger
1. Pick the coffeelog repository (that is, your fork of my repo)
1. Select the "cloudbuild.yml" option and specify the file path as that.
1. Create the trigger.
1. Trigger the first build manually.
1. See the logs if the image build succeeds.

Verify the images are pushed (the image name format should be
`gcr.io/PROJECT_ID/...`):

```sh
$ gcloud alpha container images list

NAME
gcr.io/ahmetb-starter/coffeedirectory
gcr.io/ahmetb-starter/userdirectory
gcr.io/ahmetb-starter/web
```
