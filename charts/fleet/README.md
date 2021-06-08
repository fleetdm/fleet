## Fleet Helm Chart

This directory contains a Helm Chart that makes deploying Fleet on Kubernetes easy.

### Usage

#### 1. Create namespace

This Helm chart does not auto-provision a namespace. You can add one with `kubectl create namespace <name>` or by creating a YAML file containing a service and applying it to your cluster.

#### 2. Create the necessary secrets

This Helm chart does not create the Kubernetes `Secret`s necessary for Fleet to operate. At a minimum, secrets for the MySQL password must be created. For example, if you are deploying into a namespace called `fleet`:

```yaml
---
kind: Secret
apiVersion: v1
metadata:
  name: mysql
  namespace: fleet
stringData:
  mysql-password: this-is-a-bad-password
```

If you use Fleet's TLS capabilities, TLS connections to the MySQL server, or AWS access secret keys, additional secrets and keys are needed. The name of each `Secret` must match the value of `secretName` for each section in the `values.yaml` file and the key of each secret must match the related key value from the values file. For example, to configure Fleet's TLS, you would use a Secret like the one below.

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: fleet
  namespace: fleet
stringData:
  server.cert: |
    your-pem-encoded-certificate-here
  server.key: |
    your-pem-encoded-key-here
```

Once all of your secrets are configured, use `kubectl apply -f <secret_file_name.yaml> --namespace <your_namespace>` to create them in the cluster.

#### 3. Further Configuration

To configure how Fleet runs, such as specifying the number of Fleet instances to deploy or changing the logger plugin for Fleet, edit the `values.yaml` file to your desired settings.

#### 4. Deploy Fleet

Once the secrets have been created and you have updated the values to match your required configuration, you can deploy with the following command.

```sh
helm upgrade --install fleet fleet \
  --namespace <your_namespace> \
  --repo https://fleetdm.github.io/fleet/charts \
  --values values.yaml
```
