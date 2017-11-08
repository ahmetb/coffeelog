# Deploy Network Policies to whitelist cluster networking

By default Kubernetes allows all pods to talk to all pods. [Network
Policies](https://ahmet.im/blog/kubernetes-network-policy/) help us restrict
this. While creating the cluster, we passed `--enable-network-policy` that
installed Calico Networking Plugin to the cluster.

First, we should drop all non-whitelisted connections. To achieve this, we will
have a per-namespace Deny-All policy. See
[`networkpolicy-default-deny-all.yaml`](../misc/kube/networkpolicy-default-deny-all.yaml).

Then, we create other `networkpolicy-*.yaml` files that allow these:

- `coffeedirectory` can connect to `userdirectory`
- `web` can connect to `userdirectory` and `coffeedirectory`
- everything can connect to `web` (required for external load balancers to work)

At the end we end up these Network Policies:

```sh
$ kubectl get networkpolicy
NAME                    POD-SELECTOR          AGE
coffeedirectory-allow   app=coffeedirectory   2m
default-deny-all        <none>                11m
userdirectory-allow     app=userdirectory     2m
web-allow               app=web               2m
```

created from:

- [networkpolicy-default-deny-all.yaml](../misc/kube/networkpolicy-default-deny-all.yaml)
- [networkpolicy-web.yaml](../misc/kube/networkpolicy-web.yaml)
- [networkpolicy-userdirectory.yaml](../misc/kube/networkpolicy-userdirectory.yaml)
- [networkpolicy-coffeedirectory.yaml](../misc/kube/networkpolicy-coffeedirectory.yaml)
