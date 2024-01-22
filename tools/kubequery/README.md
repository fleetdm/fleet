# Kubequery and Fleet

Use the provided configuration file ([kubequery-fleet.yml](kubequery-fleet.yml)) to get a [kubequery](../../infrastructure/kubequery) instance connected to Fleet.

Before deploying, first retrieve the enroll secret from Fleet by opening a web browser to the Fleet URL, going to the Hosts page, and clicking on the "Manage enroll secret" button.
Alternatively, you can get the enroll secret using `fleetctl` using `fleetctl get enroll-secret`.
Update the `enroll.secret` in the `ConfigMap`. In production, you will also need to update the `tls_hostname` and `fleet.pem` to the appropriate values. In order to download the `fleet.pem` certificate chain, navigate to the "Hosts> Add hosts> Advanced" tab and select "Download". Finally, deploy kubequery using `kubectl`


```sh
kubectl apply -f kubequery-fleet.yml
```

Kubernetes clusters will show up in Fleet with hostnames like `kubequery <CLUSTER NAME>`.

Sample queries are included in the configuration file ([queries-kubequery-fleet.yml](queries-kubequery-fleet.yml)). Modify the `team` value in this file to reflect the appropriate team name for your environment and apply with `fleetctl`.

```
fleetctl apply -f queries-kubequery-fleet.yml
```
