# Server installation

- [Fleet on CentOS](#fleet-on-centos)
  - [Setting up a host](#setting-up-a-host)
  - [Installing Fleet](#installing-fleet)
  - [Installing and configuring dependencies](#installing-and-configuring-dependencies)
    - [MySQL](#mysql)
    - [Redis](#redis)
  - [Running the Fleet server](#running-the-fleet-server)
  - [Running Fleet with systemd](#running-fleet-with-systemd)
  - [Installing and running osquery](#installing-and-running-osquery)
  - [Running the Fleet server](#running-the-fleet-server-1)
  - [Running Fleet with systemd](#running-fleet-with-systemd-1)
  - [Installing and running osquery](#installing-and-running-osquery-1)
- [Fleet on Kubernetes](#deploying-fleet-on-kubernetes)
  - [Installing Fleet with kubectl](#installing-fleet-with-kubectl)
  - [Installing Helm](#installing-helm)
  - [Installing Fleet with Helm](#installing-fleet-with-helm)
  - [Installing infrastructure dependencies](#installing-infrastructure-dependencies-with-helm)
    - [MySQL](#mysql-2)
    - [Redis](#redis-2)
  - [Setting up and installing Fleet](#setting-up-and-installing-Fleet)
    - [Create server secrets](#create-server-secrets)
    - [Deploying Fleet](#deploying-fleet)
    - [Deploying the load balancer](#deploying-the-load-balancer)
    - [Configure DNS](#configure-dns)
- [Fleet on AWS ECS](#deploying-fleet-on-aws-ecs)
- [Building Fleet from Source](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Building-Fleet.md)
- [Community projects](#community-projects)

## Fleet on CentOS

In this guide, we're going to install Fleet and all of its application dependencies on a CentOS 7.1 server. Once we have Fleet up and running, we're going to install osquery on that same CentOS 7.1 host and enroll it in Fleet. This should give you a good understanding of both how to install Fleet as well as how to install and configure osquery such that it can communicate with Fleet.

### Setting up a host

If you don't have a CentOS host readily available, Fleet recommends using [Vagrant](https://www.vagrantup.com/) for this guide. You can find installation instructions on Vagrant's [downloads page](https://developer.hashicorp.com/vagrant/downloads).

Once you have installed Vagrant, run the following to create a Vagrant box, start it, and log into it:

```
echo 'Vagrant.configure("2") do |config|
  config.vm.box = "bento/centos-7.1"
  config.vm.network "forwarded_port", guest: 8080, host: 8080
end' > Vagrantfile
vagrant up
vagrant ssh
```

### Installing Fleet

To install Fleet, [download](https://github.com/fleetdm/fleet/releases), unzip, and move the latest Fleet binary to your desired install location.

For example, after downloading:
```sh
unzip fleet.zip 'linux/*' -d fleet
sudo cp fleet/linux/fleet* /usr/bin/
```

### Installing and configuring dependencies

#### MySQL

To install the MySQL server files, run the following:

```
wget https://repo.mysql.com/mysql57-community-release-el7.rpm
sudo rpm -i mysql57-community-release-el7.rpm
sudo yum update
sudo yum install mysql-server
```

To start the MySQL service:

```
sudo systemctl start mysqld
```

Let's set a password for the MySQL root user.
MySQL creates an initial temporary root password which you can find in `/var/log/mysqld.log` you will need this password to change the root password.

Connect to MySQL

```
mysql -u root -p
```

When prompted enter in the temporary password from `/var/log/mysqld.log`

Change root password, in this case we will use `toor?Fl33t` as default password validation requires a more complex password.

For MySQL 5.7.6 and newer, use the following command:

```
mysql> ALTER USER "root"@"localhost" IDENTIFIED BY "toor?Fl33t";
```

For MySQL 5.7.5 and older, use:

```
mysql> SET PASSWORD FOR "root"@"localhost" = PASSWORD("toor?Fl33t");
```

Now issue the command

```
mysql> flush privileges;
```

And exit MySQL

```
mysql> exit
```

Stop MySQL and start again

```
sudo mysqld stop
sudo systemctl start mysqld
```

It's also worth creating a MySQL database for us to use at this point. Run the following to create the `fleet` database in MySQL. Note that you will be prompted for the password you created above.

```
echo 'CREATE DATABASE fleet;' | mysql -u root -p
```

#### Redis

To install the Redis server files, run the following:

```
sudo rpm -Uvh https://archives.fedoraproject.org/pub/archive/epel/6/i386/epel-release-6-8.noarch.rpm
sudo yum install redis
```

To start the Redis server in the background, you can run the following:

```
sudo service redis start
```

### Running the Fleet server

Now that we have installed Fleet, MySQL, and Redis, we are ready to launch Fleet! First, we must "prepare" the database. We do this via `fleet prepare db`:

```
/usr/bin/fleet prepare db \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=root \
  --mysql_password=toor?Fl33t
```

The output should look like:

```
Migrations completed.
```

Before we can run the server, we need to generate some TLS keying material. If you already have tooling for generating valid TLS certificates, then you are encouraged to use that instead. You will need a TLS certificate and key for running the Fleet server. If you'd like to generate self-signed certificates, you can do this via (replace SERVER_NAME with your server FQDN):

```
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout /tmp/server.key -out /tmp/server.cert -subj "/CN=SERVER_NAME” \
  -addext "subjectAltName=DNS:SERVER_NAME”
```

You should now have two new files in `/tmp`:

- `/tmp/server.cert`
- `/tmp/server.key`

Now we are ready to run the server! We do this via `fleet serve`:

```
/usr/bin/fleet serve \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=root \
  --mysql_password=toor \
  --redis_address=127.0.0.1:6379 \
  --server_cert=/tmp/server.cert \
  --server_key=/tmp/server.key \
  --logging_json
```

Now, if you go to [https://localhost:8080](https://localhost:8080) in your local browser, you should be redirected to [https://localhost:8080/setup](https://localhost:8080/setup) where you can create your first Fleet user account.

### Running Fleet with systemd

See [Running with systemd](https://fleetdm.com/docs/deploying/configuration#running-with-systemd) for documentation on running fleet as a background process and managing the fleet server logs.

### Installing and running osquery

> Note that this whole process is outlined in more detail in the [Adding Hosts To Fleet](https://fleetdm.com/docs/using-fleet/adding-hosts) document. The steps are repeated here for the sake of a continuous tutorial.

To install osquery on CentOS, you can run the following:

```
sudo rpm -ivh https://osquery-packages.s3.amazonaws.com/centos7/noarch/osquery-s3-centos7-repo-1-0.0.noarch.rpm
sudo yum install osquery
```

You will need to set the osquery enroll secret and osquery server certificate. If you head over to the manage hosts page on your Fleet instance (which should be [https://localhost:8080/hosts/manage](https://localhost:8080/hosts/manage)), you should be able to click "Add New Hosts" and see a modal like the following:

![Add New Host](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/add-new-host-modal.png)

If you select "Fetch Fleet Certificate", your browser will download the appropriate file to your downloads directory (to a file probably called `localhost-8080.pem`). Copy this file to your CentOS host at `/var/osquery/server.pem`.

You can also select "Reveal Secret" on that modal and the enrollment secret for your Fleet instance will be revealed. Copy that text and create a file with its contents:

```
echo 'LQWzGg9+/yaxxcBUMY7VruDGsJRYULw8' | sudo tee /var/osquery/enroll_secret
```

Now you're ready to run the `osqueryd` binary:

```
sudo /usr/bin/osqueryd \
  --enroll_secret_path=/var/osquery/enroll_secret \
  --tls_server_certs=/var/osquery/server.pem \
  --tls_hostname=localhost:8080 \
  --host_identifier=instance \
  --enroll_tls_endpoint=/api/osquery/enroll \
  --config_plugin=tls \
  --config_tls_endpoint=/api/osquery/config \
  --config_refresh=10 \
  --disable_distributed=false \
  --distributed_plugin=tls \
  --distributed_interval=3 \
  --distributed_tls_max_attempts=3 \
  --distributed_tls_read_endpoint=/api/osquery/distributed/read \
  --distributed_tls_write_endpoint=/api/osquery/distributed/write \
  --logger_plugin=tls \
  --logger_tls_endpoint=/api/osquery/log \
  --logger_tls_period=10
```

If you go back to [https://localhost:8080/hosts/manage](https://localhost:8080/hosts/manage), you should have a host successfully enrolled in Fleet!

---

## Deploying Fleet on Kubernetes

In this guide, we will focus on deploying Fleet only on a Kubernetes cluster. Kubernetes is a container orchestration tool that was open sourced by Google in 2014.

There are 2 primary ways to deploy the Fleet server to a Kubernetes cluster. The first is via `kubectl` with a `deployment.yml` file. The second is using Helm, the Kubernetes Package Manager.

### Deploying Fleet with kubectl

We will assume you have `kubectl` and MySQL and Redis are all set up and running. Optionally you have minikube to test your deployment locally on your machine.

To deploy the Fleet server and connect to its dependencies(MySQL and Redis), we will set up a `deployment.yml` file with the following specifications:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fleet-deployment
  labels:
    app: fleet
spec:
  replicas: 3
  selector:
    matchLabels:
      app: fleet
  template:
    metadata:
      labels:
        app: fleet
    spec:
      containers:
      - name: fleet
        image: fleetdm/fleet:4.32.0
        env:
          # if running Fleet behind external ingress controller that terminates TLS
          - name: FLEET_SERVER_TLS
            value: FALSE
          - name: FLEET_VULNERABILITIES_DATABASES_PATH
            value: /tmp/vuln
          - name: FLEET_MYSQL_ADDRESS
            valueFrom:
              secretKeyRef:
                name: fleet_secrets
                key: mysql_address
          - name: FLEET_MYSQL_DATABASE
            valueFrom:
              secretKeyRef:
                name: fleet_secrets
                key: mysql_database
          - name: FLEET_MYSQL_PASSWORD
            valueFrom:
              secretKeyRef:
                name: fleet_secrets
                key: mysql_password
          - name: FLEET_MYSQL_USERNAME
            valueFrom:
              secretKeyRef:
                name: fleet_secrets
                key: mysql_username
          - name: FLEET_REDIS_ADDRESS
            valueFrom:
              secretKeyRef:
                name: fleet_secrets
                key: redis_address
        volumeMounts:
          - name: tmp
            mountPath: /tmp # /tmp might not work on all cloud providers by default
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "2048Mi" # vulnerability processing
            cpu: "500m"
        ports:
        - containerPort: 3000
      volumes:
        - name: tmp
          emptyDir:

```
Notice we are using secrets to pass in values for Fleet's dependencies' environment variables.

Let's tell Kubernetes to create the cluster by running the below command.

`kubectl apply -f ./deployment.yml`


### Initializing Helm

If you have not used Helm before, you must run the following to initialize your cluster prior to installing Fleet:

```
helm init
```

### Deploying Fleet with Helm

To configure preferences for Fleet for use in Helm, including secret names, MySQL and Redis hostnames, and TLS certificates, download the [values.yaml](https://raw.githubusercontent.com/fleetdm/fleet/main/charts/fleet/values.yaml) and change the settings to match your configuration.

Please note you will need all dependencies configured prior to installing the Fleet Helm Chart as it will try and run database migrations immediately.

Once you have those configured, run the following:

```
helm upgrade --install fleet fleet \
  --repo https://fleetdm.github.io/fleet/charts \
  --values values.yaml
```

The Fleet Helm Chart [README.md](https://github.com/fleetdm/fleet/blob/main/charts/fleet/README.md) also includes an example using namespaces, which is outside the scope of the examples below.

### Installing infrastructure dependencies with Helm

For the sake of this tutorial, we will again use Helm, this time to install MySQL and Redis.

#### MySQL

The MySQL that we will use for this tutorial is not replicated and it is not Highly Available. If you're deploying Fleet on a Kubernetes managed by a cloud provider (GCP, Azure, AWS, etc), I suggest using their MySQL product if possible as running HA MySQL in Kubernetes can be difficult. To make this tutorial cloud provider agnostic however, we will use a non-replicated instance of MySQL.

To install MySQL from Helm, run the following command. Note that there are some options that are specified. These options basically just enumerate that:

- There should be a `fleet` database created
- The default user's username should be `fleet`

```
helm install \
  --name fleet-database \
  --set mysqlUser=fleet,mysqlDatabase=fleet \
  stable/mysql
```

This helm package will create a Kubernetes `Service` which exposes the MySQL server to the rest of the cluster on the following DNS address:

```
fleet-database-mysql:3306
```

We will use this address when we configure the Kubernetes deployment and database migration job, but if you're not using a Helm-installed MySQL in your deployment, you'll have to change this in your Kubernetes config files. For the Fleet Helm Chart, this will be used in the `values.yaml`.

##### Database migrations

Note: this step is not neccessary when using the Fleet Helm Chart as it handles migrations automatically.

The last step is to run the Fleet database migrations on your new MySQL server. To do this, run the following:

```
kubectl create -f ./docs/Using-Fleet/configuration-files/kubernetes/fleet-migrations.yml
```

In Kubernetes, you can only run a job once. If you'd like to run it again (i.e.: you'd like to run the migrations again using the same file), you must delete the job before re-creating it. To delete the job and re-run it, you can run the following commands:

```
kubectl delete -f ./docs/Using-Fleet/configuration-files/kubernetes/fleet-migrations.yml
kubectl create -f ./docs/Using-Fleet/configuration-files/kubernetes/fleet-migrations.yml
```

#### Redis

```
helm install \
  --name fleet-cache \
  --set persistence.enabled=false \
  stable/redis
```

This helm package will create a Kubernetes `Service` which exposes the Redis server to the rest of the cluster on the following DNS address:

```
fleet-cache-redis:6379
```

We will use this address when we configure the Kubernetes deployment, but if you're not using a Helm-installed Redis in your deployment, you'll have to change this in your Kubernetes config files. If you are using the Fleet Helm Chart, this will also be used in the `values.yaml` file.

### Setting up and installing Fleet

> **A note on container versions**
>
> The Kubernetes files referenced by this tutorial use the Fleet container tagged at `1.0.5`. The tag is something that should be consistent across the migration job and the deployment specification. If you use these files, I suggest creating a workflow that allows you templatize the value of this tag. For further reading on this topic, see the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/overview/#container-images).

#### Create server secrets

It should be noted that by default Kubernetes stores secret data in plaintext in etcd. Using an alternative secret storage mechanism is outside the scope of this tutorial, but let this serve as a reminder to secure the storage of your secrets.

##### TLS certificate & key

Consider using Lets Encrypt to easily generate your TLS certificate. For examples on using `lego`, the command-line Let's Encrypt client, see the [documentation](https://github.com/xenolf/lego#cli-example). Consider the following example, which may be useful if you're a GCP user:

```
GCE_PROJECT="acme-gcp-project" GCE_DOMAIN="acme-co" \
  lego --email="username@acme.co" \
    -x "http-01" \
    -x "tls-sni-01" \
    --domains="fleet.acme.co" \
    --dns="gcloud" --accept-tos run
```

If you're going the route of a more traditional CA-signed certificate, you'll have to generate a TLS key and a CSR (certificate signing request):

```
openssl req -new -newkey rsa:2048 -nodes -keyout tls.key -out tls.csr
```

Now you'll have to give this CSR to a Certificate Authority, and they will give you a file called `tls.crt`. We will then have to add the key and certificate as Kubernetes secrets.

```
kubectl create secret tls fleet-tls --key=./tls.key --cert=./tls.crt
```

#### Deploying Fleet

First we must deploy the instances of the Fleet webserver. The Fleet webserver is described using a Kubernetes deployment object. To create this deployment, run the following:

```
kubectl apply -f ./docs/Using-Fleet/configuration-files/kubernetes/fleet-deployment.yml
```

You should be able to get an instance of the webserver running via `kubectl get pods` and you should see the following logs:

```
kubectl logs fleet-webserver-9bb45dd66-zxnbq
ts=2017-11-16T02:48:38.440578433Z component=service method=ListUsers user=none err=null took=2.350435ms
ts=2017-11-16T02:48:38.441148166Z transport=https address=0.0.0.0:443 msg=listening
```

#### Deploying the load balancer

Now that the Fleet server is running on our cluster, we have to expose the Fleet webservers to the internet via a load balancer. To create a Kubernetes `Service` of type `LoadBalancer`, run the following:

```
kubectl apply -f ./docs/Using-Fleet/configuration-files/kubernetes/fleet-service.yml
```

#### Configure DNS

Finally, we must configure a DNS address for the external IP address that we now have for the Fleet load balancer. Run the following to show some high-level information about the service:

```
kubectl get services fleet-loadbalancer
```

In this output, you should see an "EXTERNAL-IP" column. If this column says `<pending>`, then give it a few minutes. Sometimes acquiring a public IP address can take a moment.

Once you have the public IP address for the load balancer, create an A record in your DNS server of choice. You should now be able to browse to your Fleet server from the internet!

---

## Deploying Fleet on AWS ECS

Terraform reference architecture can be found [here](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws)

### Infrastructure dependencies

#### MySQL

In AWS we recommend running Aurora with MySQL Engine, see [here for terraform details](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/rds.tf#L64).

#### Redis

In AWS we recommend running ElastiCache (Redis Engine) see [here for terraform details](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/redis.tf#L13)

#### Fleet server

Running Fleet in ECS consists of two main components the [ECS Service](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L84) & [Load Balancer](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L59). In our example the ALB is [handling TLS termination](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L46)

#### Fleet migrations

Migrations in ECS can be achieved (and is recommended) by running [dedicated ECS tasks](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws#migrating-the-db) that run the `fleet prepare --no-prompt=true db` command. See [terraform for more details](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L261)

Alternatively you can bake the prepare command into the same task definition see [here for a discussion](https://github.com/fleetdm/fleet/pull/1761#discussion_r697599457), but this not recommended for production environments.

---

## Community projects

Below are some projects created by Fleet community members. These projects provide additional solutions for deploying Fleet. Please submit a pull request if you'd like your project featured.

- [CptOfEvilMinions/FleetDM-Automation](https://github.com/CptOfEvilMinions/FleetDM-Automation) - Ansible and Docker code to set up Fleet

<meta name="pageOrderInSection" value="200">
<meta name="description" value="Information on installing and running the Fleet server on various platforms, including CentOS, Kubernetes, and AWS ECS.">
<meta name="navSection" value="Deployment guides">