# Kubequery and Fleet

Use the provided configuration file ([kubequery-fleet.yml](kubequery-fleet.yml)) to get a [kubequery](https://github.com/Uptycs/kubequery) instance connected to Fleet.

Outside of development environments, it will be necessary to change the `tls_hostname`, `enroll.secret`, and `fleet.pem` to the appropriate values.

Once the configuration is modified as appropriate, apply with `kubectl`:

```sh
kubectl apply -f kubequery-fleet.yml
```

Kubernetes clusters will show up in Fleet with hostnames like `kubequery <CLUSTER_NAME>`.