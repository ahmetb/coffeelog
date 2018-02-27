# Set up domain name

Find the static IP address of the Ingress and use your DNS configuration on your
nameservers to create an A record on a domain name (like `coffee.ahmet.im`) with
this IP address.

    kubectl get ingress

Using this hostname, you can go back to [API
Manager](https://console.cloud.google.com/apis/dashboard) and edit the callback
URL from `localhost` to this domain name.
