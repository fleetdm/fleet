[![Build](https://github.com/Uptycs/kubequery/workflows/Build/badge.svg?branch=master)](https://github.com/Uptycs/kubequery/actions?query=workflow%3ABuild)
[![CodeQL](https://github.com/Uptycs/kubequery/workflows/CodeQL/badge.svg?branch=master)](https://github.com/Uptycs/kubequery/actions?query=workflow%3ACodeQL)
[![Go Report Card](https://goreportcard.com/badge/github.com/Uptycs/kubequery)](https://goreportcard.com/report/github.com/Uptycs/kubequery)
[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B22616%2Fgit%40github.com%3AUptycs%2Fkubequery.git.svg?type=shield)](https://app.fossa.com/projects/custom%2B22616%2Fgit%40github.com%3AUptycs%2Fkubequery.git?ref=badge_shield) [![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](CODE_OF_CONDUCT.md)

---

# kubequery powered by Osquery

kubequery is a [Osquery](https://osquery.io) extension that provides SQL based analytics for [Kubernetes](https://kubernetes.io) clusters

kubequery will be packaged as docker image available from [dockerhub](https://hub.docker.com/r/uptycs/kubequery). It is expected to be deployed as a [Kubernetes Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment) per cluster. A sample deployment template is available [here](kubequery-template.yaml)

kubequery tables [schema is available here](docs/schema.md)

---

## Build

`Go 1.17` and `make` are required to build kubequery. Run: `make`

Container image for master branch will be available on [dockerhub](https://hub.docker.com/r/uptycs/kubequery)
```sh
docker pull uptycs/kubequery:latest
```

For production, tagged container images should be used instead of `latest`.

---

## Deployment

[kubequery-template.yaml](kubequery-template.yaml) is a template that creates the following Kubernetes resources:
* [Namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)
* [ServiceAccount](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#service-account-tokens)
* [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole)
* [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding)
* [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/)
* [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)

`kubequery` Namespace will be the placeholder for all resources that are namespaced.

`kubequery-sa` is ServiceAccount that is associated with the kubequery deployment pod specification. The container uses the service account token to authenticate with the API server.

`kubequery-clusterrole` is a ClusterRole that allows `get` and `list` operations on all resources in the following API groups:
- "" (core)
- admissionregistration&#46;k8s&#46;io
- apps
- autoscaling
- batch
- networking&#46;k8s&#46;io
- policy
- rbac&#46;authorization&#46;k8s&#46;io
- storage&#46;k8s&#46;io

`kubequery-clusterrolebinding` is a ClusterRoleBinding that binds the cluster role with the service account.

`kubequery-config` is a ConfigMap that will be mounted inside the container image as a directory. The contents of this config map should be similar to `/etc/osquery`. For example, kubequery.flags, kubequery.conf, etc. should be part of this config map.

`kubequery` is the Deployment that creates one replica pod. The container launched as a part of the pod is run as non-root user.

By default pod resource `requests` and `limits` are set to 500m (half a core) and 200MB. kubequery.yaml file should be tweaked to suite your needs before applying:

```sh
kubectl apply -f kubequery.yaml
```

Check the status of the pod using the following command. Pod should be in Running Status.
```sh
kubectl get pods -n kubequery
```

Validate the installation was successful by first executing:

```sh
kubectl exec -it $(kubectl get pods -n kubequery -o jsonpath='{.items[0].metadata.name}') -n kubequery -- kubequeryi '.tables'
```

Which should produce the following output:

```
  => kubernetes_api_resources
  => kubernetes_cluster_role_binding_subjects
  => kubernetes_cluster_role_policy_rule
  => kubernetes_config_maps
  => kubernetes_cron_jobs
  => kubernetes_csi_drivers
  => kubernetes_csi_node_drivers
  => kubernetes_daemon_set_containers
  ...
```

Queries can be run using kubequeryi on the deployed container:

```sh
kubectl exec -it $(kubectl get pods -n kubequery -o jsonpath='{.items[0].metadata.name}') -n kubequery -- kubequeryi --line 'SELECT * FROM kubernetes_pods'
```

Pod logs can be viewed using:
```sh
kubectl logs $(kubectl get pods -n kubequery -o jsonpath='{.items[0].metadata.name}') -n kubequery
```

## Helm

[Helm](https://helm.sh) must be installed to use the charts. Please refer to Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

```sh
helm repo add uptycs https://uptycs.github.io/kubequery
```

If you had already added this repo earlier, run `helm repo update` to retrieve the latest versions of the packages. You can then run `helm search repo uptycs` to see the charts.

To install the kubequery chart:
```sh
helm install my-kubequery uptycs/kubequery
```

To uninstall the chart:
```sh
helm delete my-kubequery
```

---

## FAQ

### Use kubequery instead of Osquery in Kubernetes?

No. kubequery should to be deployed as a [Kubernetes Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/). Which means there will be one [Pod](https://kubernetes.io/docs/concepts/workloads/pods/) of kubequery running per Kubernetes cluster. Osquery should be deployed to every node in the cluster. Querying most Osquery tables from an ephemeral pod does not provide much value. kubequery container image also runs as non-root user, which means most of the Osquery tables will either return an error or partial data.

![Deployment](docs/deployment.svg)

### Why are some columns JSON?

Normalizing nested JSON data like Kubernetes API responses will create an explosion of tables. So some of the columns in kuberenetes tables are left as JSON. Data is eventually processed by [SQLite](https://www.sqlite.org/index.html) with-in Osquery. SQLite has very [good JSON](https://www.sqlite.org/json1.html) support.

For example if `run_as_user` in `kubernetes_pod_security_policies` table looks like the following:
```json
{"rule": "MustRunAsNonRoot"}
```

To get the value of `rule`, the following query can be used:
```sql
SELECT value AS 'rule'
FROM kubernetes_pod_security_policies, json_tree(kubernetes_pod_security_policies.run_as_user)
WHERE key = 'rule';

+------------------+
| rule             |
+------------------+
| MustRunAsNonRoot |
+------------------+
```

[json_each](https://www.sqlite.org/json1.html#jeach) can be used to explode JSON array types. For example if `volumes` in `kubernetes_pod_security_policies` table looks like the following:
```json
{"volumes": ["configMap","emptyDir","projected","secret","downwardAPI","persistentVolumeClaim"]}
```

To get a separate row for each volume, the following query can be used:
```sql
SELECT value
FROM kubernetes_pod_security_policies, json_each(kubernetes_pod_security_policies.volumes);

+-----------------------+
| value                 |
+-----------------------+
| configMap             |
| emptyDir              |
| projected             |
| secret                |
| downwardAPI           |
| persistentVolumeClaim |
+-----------------------+
```

Osquery logger's like TLS, Kafka loggers can be used to export scheduled query data to remove fleet management/security analytics platforms. Lamba like functions can be applied on rows of streaming data in these platforms. These lamba functions can extract necessary fields from embedded JSON to detect compliance issues or security concerns. If tables are normalized and are streamed at different schedules, it will not be trivial to JOIN across tables and trigger events/alerts.
