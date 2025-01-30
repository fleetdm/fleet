# Deploy Fleet on Kubernetes with Helm

> **Archived.** While still usable, this guide has not been updated recently. See the [Deploy Fleet](https://fleetdm.com/docs/deploy/deploy-fleet) docs for supported deployment methods.

![Deploy Fleet on Kubernetes](../website/assets/images/articles/deploy-fleet-on-kubernetes-800x450@2x.png)

> Updated on January 28, 2025, by [Noah Talerman](https://github.com/noahtalerman).

In this guide, we will focus on deploying Fleet only on a Kubernetes cluster. Kubernetes is a container orchestration tool that was open sourced by Google in 2014.

## Initializing Helm

If you have not used Helm before, you must run the following to initialize your cluster prior to installing Fleet:

```sh
helm init
```

> Note: The helm init command has been removed in Helm v3. It performed two primary functions. First, it installed Tiller which is no longer needed. Second, it set up directories and repositories where Helm configuration lived. This is now automated in Helm v3; if the directory is not present it will be created.

## Deploying Fleet with Helm

To configure preferences for Fleet for use in Helm, including secret names, MySQL and Redis hostnames, and TLS certificates, download the [values.yaml](https://raw.githubusercontent.com/fleetdm/fleet/main/charts/fleet/values.yaml) and change the settings to match your configuration.

Please note you will need all dependencies configured prior to installing the Fleet Helm Chart as it will try and run database migrations immediately.

Once you have those configured, run the following:

```sh
helm upgrade --install fleet fleet \
  --repo https://fleetdm.github.io/fleet/charts \
  --values values.yaml
```

The Fleet Helm Chart [README.md](https://github.com/fleetdm/fleet/blob/main/charts/fleet/README.md) also includes an example using namespaces, which is outside the scope of the examples below.

## Installing infrastructure dependencies with Helm

### MySQL

The MySQL that we will use for this tutorial is not replicated and it is not Highly Available. If you're deploying Fleet on a Kubernetes managed by a cloud provider (GCP, Azure, AWS, etc), I suggest using their MySQL product if possible as running HA MySQL in Kubernetes can be difficult. To make this tutorial cloud provider agnostic however, we will use a non-replicated instance of MySQL.

To install MySQL from Helm, run the following command. Note that there are some options that need to be defined:

- There should be a `fleet` database created
- The default user's username should be `fleet`

Helm v2
```sh
helm install \
  --name fleet-database \
  --set auth.username=fleet,auth.database=fleet \
  oci://registry-1.docker.io/bitnamicharts/mysql
```

Helm v3
```sh
helm install fleet-database \
  --set auth.username=fleet,auth.database=fleet \
  oci://registry-1.docker.io/bitnamicharts/mysql 
```

This helm package will create a Kubernetes `Service` which exposes the MySQL server to the rest of the cluster on the following DNS address:

```
fleet-database-mysql:3306
```

We will use this address when we configure the Kubernetes deployment and database migration job, but if you're not using a Helm-installed MySQL in your deployment, you'll have to change this in your Kubernetes config files. For the Fleet Helm Chart, this will be used in the `values.yaml`.

### Redis

Helm v2
```sh
helm install \
  --name fleet-cache \
  --set persistence.enabled=false \
  oci://registry-1.docker.io/bitnamicharts/redis
```

Helm v3
```sh
helm install fleet-cache \
  --set persistence.enabled=false \
  oci://registry-1.docker.io/bitnamicharts/redis
```

This helm package will create a Kubernetes `Service` which exposes the Redis server to the rest of the cluster on the following DNS address:

```
fleet-cache-redis:6379
```

We will use this address when we configure the Kubernetes deployment, but if you're not using a Helm-installed Redis in your deployment, you'll have to change this in your Kubernetes config files. If you are using the Fleet Helm Chart, this will also be used in the `values.yaml` file.

<meta name="articleTitle" value="Deploy Fleet on Kubernetes with Helm">
<meta name="authorGitHubUsername" value="marpaia">
<meta name="authorFullName" value="Mike Arpaia">
<meta name="publishedOn" value="2017-11-18">
<meta name="category" value="guides">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-fleet-on-kubernetes-800x450@2x.png">
<meta name="description" value="Learn how to deploy Fleet on Kubernetes using Helm.">
