# Example deployment scenarios

- [Fleet on CentOS](#fleet-on-centos)
  - [Setting up a host](#setting-up-a-host)
  - [Installing Fleet](#installing-fleet)
  - [Installing and configuring dependencies](#installing-and-configuring-dependencies)
    - [MySQL](#mysql)
    - [Redis](#redis)
  - [Running the Fleet server](#running-the-fleet-server)
  - [Running Fleet with systemd](#running-fleet-with-systemd)
  - [Installing and running osquery](#installing-and-running-osquery)
- [Fleet on Ubuntu](#fleet-on-ubuntu)
  - [Setting up a host](#setting-up-a-host-1)
  - [Installing Fleet](#installing-fleet-1)
  - [Installing and configuring dependencies](#installing-and-configuring-dependencies-1)
    - [MySQL](#mysql-1)
    - [Redis](#redis-1)
  - [Running the Fleet server](#running-the-fleet-server-1)
  - [Running Fleet with systemd](#running-fleet-with-systemd-1)
  - [Installing and running osquery](#installing-and-running-osquery-1)
- [Deploying Fleet on Kubernetes](#deploying-fleet-on-kubernetes)
  - [Installing infrastructure dependencies](#installing-infrastructure-dependencies)
    - [MySQL](#mysql-2)
    - [Redis](#redis-2)
  - [Setting up and installing Fleet](#setting-up-and-installing-Fleet)
    - [Create server secrets](#create-server-secrets)
    - [Deploying Fleet](#deploying-fleet)
    - [Deploying the load balancer](#deploying-the-load-balancer)
    - [Configure DNS](#configure-dns)
- [Community projects](#community-projects)

## Fleet on CentOS

In this guide, we're going to install Fleet and all of its application dependencies on a CentOS 7.1 server. Once we have Fleet up and running, we're going to install osquery on that same CentOS 7.1 host and enroll it in Fleet. This should give you a good understanding of both how to install Fleet as well as how to install and configure osquery such that it can communicate with Fleet.

### Setting up a host

Acquiring a CentOS host to use for this guide is largely an exercise for the reader. If you don't have an CentOS host readily available, feel free to use [Vagrant](https://www.vagrantup.com/). In a clean, temporary directory, you can run the following to create a vagrant box, start it, and log into it:

```
echo 'Vagrant.configure("2") do |config|
  config.vm.box = "bento/centos-7.1"
  config.vm.network "forwarded_port", guest: 8080, host: 8080
end' > Vagrantfile
vagrant up
vagrant ssh
```

### Installing Fleet

To [install Fleet](https://github.com/fleetdm/fleet/blob/main/docs/2-Deploying/1-Installation.md), download, unzip, and move the latest Fleet binary to your desired install location.

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
sudo rpm -Uvh http://dl.fedoraproject.org/pub/epel/6/i386/epel-release-6-8.noarch.rpm
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

Before we can run the server, we need to generate some TLS keying material. If you already have tooling for generating valid TLS certificates, then you are encouraged to use that instead. You will need a TLS certificate and key for running the Fleet server. If you'd like to generate self-signed certificates, you can do this via:

```
openssl genrsa -out /tmp/server.key 4096
openssl req -new -key /tmp/server.key -out /tmp/server.csr
openssl x509 -req -days 366 -in /tmp/server.csr -signkey /tmp/server.key -out /tmp/server.cert
```

You should now have three new files in `/tmp`:

- `/tmp/server.cert`
- `/tmp/server.key`
- `/tmp/server.csr`

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

See [Running with systemd](./2-Configuration.md#running-with-systemd) for documentation on running fleet as a background process and managing the fleet server logs.

### Installing and running osquery

> Note that this whole process is outlined in more detail in the [Adding Hosts To Fleet](../1-Using-Fleet/4-Adding-hosts.md) document. The steps are repeated here for the sake of a continuous tutorial.

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
  --enroll_tls_endpoint=/api/v1/osquery/enroll \
  --config_plugin=tls \
  --config_tls_endpoint=/api/v1/osquery/config \
  --config_refresh=10 \
  --disable_distributed=false \
  --distributed_plugin=tls \
  --distributed_interval=3 \
  --distributed_tls_max_attempts=3 \
  --distributed_tls_read_endpoint=/api/v1/osquery/distributed/read \
  --distributed_tls_write_endpoint=/api/v1/osquery/distributed/write \
  --logger_plugin=tls \
  --logger_tls_endpoint=/api/v1/osquery/log \
  --logger_tls_period=10
```

If you go back to [https://localhost:8080/hosts/manage](https://localhost:8080/hosts/manage), you should have a host successfully enrolled in Fleet!

## Fleet on Ubuntu

In this guide, we're going to install Fleet and all of its application dependencies on an Ubuntu 16.04 LTS server. Once we have Fleet up and running, we're going to install osquery on that same Ubuntu 16.04 host and enroll it in Fleet. This should give you a good understanding of both how to install Fleet as well as how to install and configure osquery such that it can communicate with Fleet.

### Setting up a host

Acquiring an Ubuntu host to use for this guide is largely an exercise for the reader. If you don't have an Ubuntu host readily available, feel free to use [Vagrant](https://www.vagrantup.com/). In a clean, temporary directory, you can run the following to create a vagrant box, start it, and log into it:

```
echo 'Vagrant.configure("2") do |config|
  config.vm.box = "bento/ubuntu-16.04"
  config.vm.network "forwarded_port", guest: 8080, host: 8080
end' > Vagrantfile
vagrant up
vagrant ssh
```

### Installing Fleet

To install Fleet, run the following:

```
wget https://github.com/fleetdm/fleet/releases/latest/download/fleet.zip
unzip fleet.zip 'linux/*' -d fleet
sudo cp fleet/linux/fleet /usr/bin/fleet
sudo cp fleet/linux/fleetctl /usr/bin/fleetctl
```

### Installing and configuring dependencies

#### MySQL

To install the MySQL server files, run the following:

```
sudo apt-get install mysql-server -y
```

When asked for MySQL's root password, enter `toor` for the sake of this tutorial if you are having trouble thinking of a better password for the MySQL root user. If you decide to set your own password, be mindful that you will need to substitute it every time `toor` is used in this document.

After installing `mysql-server`, the `mysqld` server should be running. You can verify this by running the following:

```
ps aux | grep mysqld
mysql    13158  3.1 14.4 1105320 146408 ?      Ssl  21:36   0:00 /usr/sbin/mysqld
```

It's also worth creating a MySQL database for us to use at this point. Run the following to create the `fleet` database in MySQL. Note that you will be prompted for the password you created above.

```
echo 'CREATE DATABASE fleet;' | mysql -u root -p
```

#### Redis

To install the Redis server files, run the following:

```
sudo apt-get install redis-server -y
```

To start the Redis server in the background, you can run the following:

```
sudo redis-server &
```

Note that this isn't a very robust way to run a Redis server. Digital Ocean has written a very nice [community tutorial](https://www.digitalocean.com/community/tutorials/how-to-install-and-configure-redis-on-ubuntu-16-04) on installing and running Redis in a more productionalized way.

### Running the Fleet server

Now that we have installed Fleet, MySQL, and Redis, we are ready to launch Fleet! First, we must "prepare" the database. We do this via `fleet prepare db`:

```
/usr/bin/fleet prepare db \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=root \
  --mysql_password=toor
```

The output should look like:

`Migrations completed`

Before we can run the server, we need to generate some TLS keying material. If you already have tooling for generating valid TLS certificates, then you are encouraged to use that instead. You will need a TLS certificate and key for running the Fleet server. If you'd like to generate self-signed certificates, you can do this via the following steps (note - you will be asked for several bits of information, including name, contact info, and location, in order to generate the certificate):

```
openssl genrsa -out /tmp/server.key 4096
openssl req -new -key /tmp/server.key -out /tmp/server.csr
openssl x509 -req -days 366 -in /tmp/server.csr -signkey /tmp/server.key -out /tmp/server.cert
```

You should now have three new files in `/tmp`:

- `/tmp/server.cert`
- `/tmp/server.key`
- `/tmp/server.csr`

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

See [Running with systemd](./2-Configuration.md#running-with-systemd) for documentation on running fleet as a background process and managing the fleet server logs.

### Installing and running osquery

> Note that this whole process is outlined in more detail in the [Adding Hosts To Fleet](../1-Using-Fleet/4-Adding-hosts.md) document. The steps are repeated here for the sake of a continuous tutorial.

To install osquery on Ubuntu, you can run the following:

```
export OSQUERY_KEY=1484120AC4E9F8A1A577AEEE97A80C63C9D8B80B
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys $OSQUERY_KEY
sudo add-apt-repository 'deb [arch=amd64] https://pkg.osquery.io/deb deb main'
sudo apt-get update
sudo apt-get install osquery
```

If you're having trouble with the above steps, check the official [downloads](https://osquery.io/downloads) link for a direct download of the .deb.

You will need to set the osquery enroll secret and osquery server certificate. If you head over to the manage hosts page on your Fleet instance (which should be [https://localhost:8080/hosts/manage](https://localhost:8080/hosts/manage)), you should be able to click "Add New Hosts" and see a modal like the following:

![Add New Host](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/add-new-host-modal.png)

If you select "Fetch Fleet Certificate", your browser will download the appropriate file to your downloads directory (to a file probably called `localhost-8080.pem`). Copy this file to your Ubuntu host at `/var/osquery/server.pem`.

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
  --enroll_tls_endpoint=/api/v1/osquery/enroll \
  --config_plugin=tls \
  --config_tls_endpoint=/api/v1/osquery/config \
  --config_refresh=10 \
  --disable_distributed=false \
  --distributed_plugin=tls \
  --distributed_interval=3 \
  --distributed_tls_max_attempts=3 \
  --distributed_tls_read_endpoint=/api/v1/osquery/distributed/read \
  --distributed_tls_write_endpoint=/api/v1/osquery/distributed/write \
  --logger_plugin=tls \
  --logger_tls_endpoint=/api/v1/osquery/log \
  --logger_tls_period=10
```

If you go back to [https://localhost:8080/hosts/manage](https://localhost:8080/hosts/manage), you should have a host successfully enrolled in Fleet!

## Deploying Fleet on Kubernetes

In this guide, we're going to install Fleet and all of its application dependencies on a Kubernetes cluster. Kubernetes is a container orchestration tool that was open sourced by Google in 2014.

### Installing infrastructure dependencies

For the sake of this tutorial, we will use Helm, the Kubernetes Package Manager, to install MySQL and Redis. If you would like to use Helm and you've never used it before, you must run the following to initialize your cluster:

```
helm init
```

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

We will use this address when we configure the Kubernetes deployment and database migration job, but if you're not using a Helm-installed MySQL in your deployment, you'll have to change this in your Kubernetes config files.

##### Database Migrations

The last step is to run the Fleet database migrations on your new MySQL server. To do this, run the following:

```
kubectl create -f ./docs/1-Using-Fleet/configuration-files/kubernetes/fleet-migrations.yml
```

In Kubernetes, you can only run a job once. If you'd like to run it again (i.e.: you'd like to run the migrations again using the same file), you must delete the job before re-creating it. To delete the job and re-run it, you can run the following commands:

```
kubectl delete -f ./docs/1-Using-Fleet/configuration-files/kubernetes/fleet-migrations.yml
kubectl create -f ./docs/1-Using-Fleet/configuration-files/kubernetes/fleet-migrations.yml
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

We will use this address when we configure the Kubernetes deployment, but if you're not using a Helm-installed Redis in your deployment, you'll have to change this in your Kubernetes config files.

### Setting up and installing Fleet

> #### A note on container versions
>
> The Kubernetes files referenced by this tutorial use the Fleet container tagged at `1.0.5`. The tag is something that should be consistent across the migration job and the deployment specification. If you use these files, I suggest creating a workflow that allows you templatize the value of this tag. For further reading on this topic, see the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/overview/#container-images).

#### Create server secrets

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
kubectl apply -f ./docs/1-Using-Fleet/configuration-files/kubernetes/fleet-deployment.yml
```

You should be able to get an instance of the webserver running via `kubectl get pods` and you should see the following logs:

```
kubectl logs fleet-webserver-9bb45dd66-zxnbq
ts=2017-11-16T02:48:38.440578433Z component=service method=ListUsers user=none err=null took=2.350435ms
ts=2017-11-16T02:48:38.441148166Z transport=https address=0.0.0.0:443 msg=listening
```

#### Deploying the Load Balancer

Now that the Fleet server is running on our cluster, we have to expose the Fleet webservers to the internet via a load balancer. To create a Kubernetes `Service` of type `LoadBalancer`, run the following:

```
kubectl apply -f ./docs/1-Using-Fleet/configuration-files/kubernetes/fleet-service.yml
```

#### Configure DNS

Finally, we must configure a DNS address for the external IP address that we now have for the Fleet load balancer. Run the following to show some high-level information about the service:

```
kubectl get services fleet-loadbalancer
```

In this output, you should see an "EXTERNAL-IP" column. If this column says `<pending>`, then give it a few minutes. Sometimes acquiring a public IP address can take a moment.

Once you have the public IP address for the load balancer, create an A record in your DNS server of choice. You should now be able to browse to your Fleet server from the internet!

#### Community projects

Below are some projects created by Fleet community members. These projects provide additional solutions for deploying Fleet. Please submit a pull request if you'd like your project featured.

- [CptOfEvilMinions/FleetDM-Automation](https://github.com/CptOfEvilMinions/FleetDM-Automation) - Ansible and Docker code to setup FleetDM