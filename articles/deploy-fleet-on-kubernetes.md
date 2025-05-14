# Deploy Fleet on Kubernetes

> **Archived -** While still usable, this guide has not been updated recently. See the [Deploy Fleet](https://fleetdm.com/docs/deploy/deploy-fleet) docs for supported deployment methods.

> **Updated -** May 8th, 2025, by [Jorge Falcon](https://github.com/BCTBB).

![Deploy Fleet on Kubernetes](../website/assets/images/articles/deploy-fleet-on-kubernetes-800x450@2x.png)

In this guide, we will focus on deploying Fleet only on a Kubernetes cluster. This guide has been written and tested using k3s, but should function on self-hosted Kubernetes, Lightweight Kubernetes, or managed Kubernetes offerings.

## Getting Started

> You will need to have **Helm (v3)** and/or **Terraform (v1.10.2**).

Before we get started with deploying Fleet through Helm or Terraform you will need access to a Kubernetes cluster, MySQL database, and Redis cluster. We will also need to create some secrets to instruct Fleet on how to connect to MySQL and Redis.

Ensure a namespace is created or already exists for your Fleet deployment resources. Example of creating a kubernetes namespace - replace `<namespace>` with the name of the namespace you'd like to use.

```sh
kubectl create ns <namespace>
```

## Installing Infrastructure Dependencies with Helm

### MySQL

> Skip if you already have a MySQL database that you plan on using.

The MySQL that we will use for this tutorial is not replicated and it is not Highly Available. If you're deploying Fleet on a Kubernetes managed by a cloud provider (GCP, Azure, AWS, etc), I suggest using their MySQL product if possible as running HA MySQL in Kubernetes can be complex. To make this tutorial cloud provider agnostic however, we will use a non-replicated instance of MySQL.

To install MySQL from Helm, run the following command. Note that there are some options that need to be defined:

- There should be a `fleet` database created
- The default user's username should be `fleet`

```sh
helm install fleet-database \
  --namespace <namespace> \
  --set auth.username=fleet,auth.database=fleet,auth.password=<password> \
  oci://registry-1.docker.io/bitnamicharts/mysql 
```

This helm package will create a Kubernetes `Service` which exposes the MySQL server to the rest of the cluster on the following DNS address:

```
fleet-database-mysql:3306
```

We will use this address when we configure the Kubernetes deployment and database migration job, but if you're not using a Helm-installed MySQL in your deployment, you'll have to change this in your Kubernetes config files. 
- For the Fleet Helm Chart, this will be used in the `values.yaml`
- For Terraform, this will be used in `main.tf`.

### Redis

> Skip if you already have a Redis cluster that you plan on using.

```sh
helm install fleet-cache \
  --namespace <namespace> \
  --set persistence.enabled=false \
  oci://registry-1.docker.io/bitnamicharts/redis
```

This helm package will create a Kubernetes `Service` which exposes the Redis server to the rest of the cluster on the following DNS address:

```
fleet-cache-redis-headless:6379
```

We will use this address when we configure the Kubernetes deployment, but if you're not using a Helm-installed Redis in your deployment, you'll have to change this in your Kubernetes config files. 
- For the Fleet Helm Chart, this will be used in the `values.yaml`
- For Terraform, this will be used in `main.tf`.

## Secrets

As noted earlier in the guide, the Fleet deployment needs to know the credentials to connect to MySQL and Redis. The secrets need to be created in the namespace that you will be deploying Fleet to. Below is an example of the body of a Kubernetes manifest to create various secrets.

To apply the secrets, you can modify the examples below as necessary and deploy to your Kubernetes cluster.

```sh
kubectl apply -f <example-manifest.yml>
```

### MySQL and Redis

- Your mysql password will be what you set in the helm install command
- Your redis password can be retrieved with the following command `kubectl get secret --namespace <namespace> fleet-cache-redis -o jsonpath="{.data.redis-password}" | base64 -d`

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: redis
  namespace: <namespace>
type: kubernetes.io/basic-auth
stringData:
  password: <redis-password-here>
---
apiVersion: v1
kind: Secret
metadata:
  name: mysql
  namespace: <namespace>
type: kubernetes.io/basic-auth
stringData:
  password: <mysql-password-here>
```

### TLS Certificates (nginx)

If you're going to be terminating TLS at your ingress (nginx) through either the Fleet Helm chart or Terraform, we'll need to create that secret.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: chart-example-tls
  namespace: <namespace>
type: kubernetes.io/tls
data:
  tls.crt: |
    <base64-encoded-tls-crt>
  tls.key: |
    <base64-encoded-tls-key>
```

## Deployment

While the examples below support ingress settings, they are limited to nginx. If you or your organization would like to use a specific ingress controller, they can be configured to handle and route traffic to the Fleet pods.

### Helm

To configure preferences for Fleet for use in Helm, including secret names, MySQL and Redis hostnames, and TLS certificates, download the [values.yaml](https://raw.githubusercontent.com/fleetdm/fleet/main/charts/fleet/values.yaml) and change the settings to match your configuration.

Please note you will need all dependencies configured prior to installing the Fleet Helm Chart as it will try and run database migrations immediately.

Once you have those configured, run the following:

```sh
helm upgrade --install fleet fleet \
  --repo https://fleetdm.github.io/fleet/charts \
  --namespace <namespace> \
  --values values.yaml
```

### Terraform

Let's start by cloning the [fleet-terraform repository](https://github.com/fleetdm/fleet-terraform).

To configure preferences for Fleet for use in Terraform, including secret names, including secret names, MySQL and Redis hostnames, and TLS certificates, we'll be modifying `fleet-terraform/k8s/example/main.tf`.

Before you can leverage the Terraform module, you will need to modify the `main.tf` with your configuration preferences for Fleet and `provider.tf` with your KUBECONFIG details for authentication. The following [link to the kubernetes provider terraform docs](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/guides/getting-started.html) has examples documented for AWS EKS, GCP GKE, and Azure.

```
provider "kubernetes" {
  # config_path = "/path/to/kubeconfig"
  config_path = ""
}
```

Once you have those configured, run the following:

1. If you have not used Terraform before, you must run the following to initialize your Terraform prior to installing Fleet:

  ```sh
  terraform init
  ```

2. To dry-run the Terraform deployment and see resources that Terraform believes will be deployed:

  ```sh
  terraform plan
  ```

3. If you're happy with the results returned by the Terraform plan, you can apply the deployment:

  ```sh
  terraform apply
  ```

I have a published [README.md](https://github.com/fleetdm/fleet-terraform/blob/tf-fleetk8s-support/k8s/README.md) with additional information and examples related to Fleet Kubernetes deployments through Terraform.

## Verify the Deployment

You can verify the status of your Fleet deployment, whether it was deployed with Helm or Terraform, by checking the status of the Kubernetes resources.

```sh
kubectl get deploy -n <namespace>
kubectl get pods -n <namespace>
```

If your Fleet deployment was successful, you should be able to access fleet with the URL that you configured `https://fleet.localhost.local`.

## Fleet Upgrades

Fleet requires that there be no active connections to the MySQL Fleet database, prior to initializing a deployment, as Database migrations are often included and risk failing. Below are instructions that can be followed to Upgrade Fleet using Helm or Terraform

#### Helm

If you've deployed Fleet with Helm, prior to an upgrade, you will need to update your `values.yml` to update the `imageTag` to be a newer version of a Fleet container image tag. Afterwards, you will need to make sure no Fleet pods are running.

```sh
kubectl scale -n <namespace> --replicas 0 deploy/fleet
```

When the Fleet `deployment` has been reduced to 0 running pods, you can proceed to upgrading Fleet.

```sh
helm upgrade --install fleet fleet \
  --repo https://fleetdm.github.io/fleet/charts \ 
  --namespace <namespace> \
  --values values.yaml
```

#### Terraform

If you've deployed Fleet with Terraform, prior to an upgrade, you will need to update your `main.tf` to update the `image_tag` to be a newer version of Fleet container image tag. Afterwards, you can initiate a Terraform apply instructing Terraform to also initiate a database migration.

```sh
terraform apply -replace=module.fleet.kubernetes_deployment.fleet
```

<meta name="articleTitle" value="Deploy Fleet on Kubernetes">
<meta name="authorGitHubUsername" value="BCTBB">
<meta name="authorFullName" value="Jorge Falcon">
<meta name="publishedOn" value="2017-11-18">
<meta name="category" value="guides">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-fleet-on-kubernetes-800x450@2x.png">
<meta name="description" value="Learn how to deploy Fleet on Kubernetes.">
