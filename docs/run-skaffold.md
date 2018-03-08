# Running locally on Minikube with Skaffold

[Skaffold] lets you
automatically deploy the code to a local
[minikube](https://github.com/kubernetes/minikube/) instance as you change and
save source code.

Install [Skaffold], start Minikube.

Now run these in separate terminal windows:

    skaffold dev -f skaffold-web.yaml

    skaffold dev -f skaffold-coffeedirectory.yaml

    skaffold dev -f skaffold-userdirectory.yaml

Then as you update the code, it will automatically rebuild and redeploy your
images to Minikube.

[Skaffold]: https://github.com/GoogleCloudPlatform/skaffold
