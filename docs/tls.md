## Set up TLS with Let’s Encrypt

Make sure you [set up a domain](domain-name.md) first.

Then follow the guide at https://github.com/ahmetb/gke-letsencrypt to install
cert-manager (the Let's Encrypt add-on) to your cluster using Helm.

Update the [certificate.yaml](../misc/kube-tls/certificate.yaml) and
[ingress-tls.yaml](../misc/kube-tls/ingress-tls.yaml) with the correct domain
name, then apply:

    kubectl apply -f ./misc/kube-tls

After a few minutes, the Ingress should be serving https:// traffic.
