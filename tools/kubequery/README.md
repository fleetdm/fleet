# Kubequery and Fleet

Use the provided configuration file ([kubequery-fleet.yml](kubequery-fleet.yml)) to get a [kubequery](https://github.com/fleetdm/kubequery) instance connected to Fleet.

Before deploying, first retrieve the enroll secret from Fleet by opening a web browser to the Fleet URL, going to the Hosts page, and clicking on the "Manage enroll secret" button.
Alternatively, you can get the enroll secret using `fleetctl` using `fleetctl get enroll-secret`.
Update the `enroll.secret` in the `ConfigMap`. In production, you will also need to update the `tls_hostname` and `fleet.pem` to the appropriate values. Finally, deploy kubequery using `kubectl`


```sh
kubectl apply -f kubequery-fleet.yml
```

Kubernetes clusters will show up in Fleet with hostnames like `kubequery <CLUSTER NAME>`.
