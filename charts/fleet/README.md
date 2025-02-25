## Fleet Helm Chart

This directory contains a Helm Chart that makes deploying Fleet on Kubernetes easy.

### Usage

#### 1. Create namespace

This Helm chart does not auto-provision a namespace. You can add one with `kubectl create namespace <name>` or by creating a YAML file containing a service and applying it to your cluster.

#### 2. Create the necessary secrets

This Helm chart does not create the Kubernetes `Secret`s necessary for Fleet to operate. You'll need to create secrets for both MySQL and Redis. For example, if you are deploying into a namespace called `fleet`:

```yaml
# MySQL secret
kind: Secret
apiVersion: v1
metadata:
  name: mysql
  namespace: fleet
stringData:
  mysql-password: your-mysql-password
  mysql-root-password: your-mysql-root-password

---
# Redis secret
kind: Secret
apiVersion: v1
metadata:
  name: redis
  namespace: fleet
stringData:
  redis-password: your-redis-password
```

The secret names and keys must match the values specified in your `values.yaml` file:
- For MySQL: `database.secretName` and `mysql.auth.existingSecret`
- For Redis: `cache.secretName` and `redis.auth.existingSecret`

If you use Fleet's TLS capabilities, you'll need additional secrets. For example, to configure Fleet's TLS:

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

#### 3. Configuration

The chart includes built-in MySQL and Redis deployments that can be enabled through the `values.yaml` file. Key configuration options include:

##### Database Configuration
```yaml
mysql:
  enabled: true  # Set to true to use the built-in MySQL
  auth:
    database: fleet
    username: fleet
    existingSecret: mysql
    passwordKey: mysql-password
    rootPasswordKey: mysql-root-password
    createDatabase: true

database:
  secretName: mysql
  address: fleet-mysql-headless:3306  # Use this address for the built-in MySQL
  database: fleet
  username: fleet
  passwordKey: mysql-password
```

##### Redis Configuration
```yaml
redis:
  enabled: true  # Set to true to use the built-in Redis
  auth:
    enabled: true
    existingSecret: redis
    existingSecretPasswordKey: redis-password
  replica:
    replicaCount: 3  # Number of Redis replicas

cache:
  address: fleet-redis-master:6379  # Use this address for the built-in Redis
  database: "0"
  usePassword: true
  secretName: redis
  passwordKey: redis-password
```

To configure how Fleet runs, such as specifying the number of Fleet instances to deploy or changing the logger plugin for Fleet, edit the `values.yaml` file to your desired settings.

#### 4. Deploy Fleet

Once the secrets have been created and you have updated the values to match your required configuration, you can deploy with the following command:

```sh
helm upgrade --install fleet fleet \
  --namespace <your_namespace> \
  --repo https://fleetdm.github.io/fleet/charts \
  --values values.yaml
```
