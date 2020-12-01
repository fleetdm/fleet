# Deploying Fleet on Kubernetes

In this guide, we're going to install Fleet and all of it's application dependencies on a Kubernetes cluster. Kubernetes is a container orchestration tool that was open sourced by Google in 2014.

## Installing Infrastructure Dependencies

For the sake of this tutorial, we will use Helm, the Kubernetes Package Manager, to install MySQL and Redis. If you would like to use Helm and you've never used it before, you must run the following to initialize your cluster:

```
helm init
```

### MySQL

The MySQL that we will use for this tutorial is not replicated and it is not Highly Available. If you're deploying Fleet on a Kubernetes managed by a cloud provider (GCP, Azure, AWS, etc), I suggest using their MySQL product if possible as running HA MySQL in Kubernetes can be difficult. To make this tutorial cloud provider agnostic however, we will use a non-replicated instance of MySQL.

To install MySQL from Helm, run the following command. Note that there are some options that are specified. These options basically just enumerate that:

- There should be a `kolide` database created
- The default user's username should be `kolide`

```
helm install \
  --name fleet-database \
  --set mysqlUser=kolide,mysqlDatabase=kolide \
  stable/mysql
```

This helm package will create a Kubernetes `Service` which exposes the MySQL server to the rest of the cluster on the following DNS address:

```
fleet-database-mysql:3306
```

We will use this address when we configure the Kubernetes deployment and database migration job, but if you're not using a Helm-installed MySQL in your deployment, you'll have to change this in your Kubernetes config files.

#### Database Migrations

The last step is to run the Fleet database migrations on your new MySQL server. To do this, run the following:

```
kubectl create -f ./examples/kubernetes/fleet-migrations.yml
```

In Kubernetes, you can only run a job once. If you'd like to run it again (i.e.: you'd like to run the migrations again using the same file), you must delete the job before re-creating it. To delete the job and re-run it, you can run the following commands:

```
kubectl delete -f ./examples/kubernetes/fleet-migrations.yml
kubectl create -f ./examples/kubernetes/fleet-migrations.yml
```

### Redis

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

## Setting Up and Installing Fleet

> ### A note on container versions
>
> The Kubernetes files referenced by this tutorial use the Fleet container tagged at `1.0.5`. The tag is something that should be consistent across the migration job and the deployment specification. If you use these files, I suggest creating a workflow that allows you templatize the value of this tag. For further reading on this topic, see the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/overview/#container-images).

### Create Server Secrets

It should be noted that by default Kubernetes stores secret data in plaintext in etcd. Using an alternative secret storage mechanism is outside the scope of this tutorial, but let this serve as a reminder to secure the storage of your secrets.

#### TLS Certificate & Key

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

#### JWT Auth Key

This is the key that will be used to sign and validate the JWT tokens that are used as the web session key.

```
echo -n "some really secure string" > ./build/fleet-server-auth-key
kubectl create secret generic fleet-server-auth-key --from-file=./build/fleet-server-auth-key
```

### Deploying Fleet

First we must deploy the instances of the Fleet webserver. The Fleet webserver is described using a Kubernetes deployment object. To create this deployment, run the following:

```
kubectl apply -f ./examples/kubernetes/fleet-deployment.yml
```

You should be able to get an instance of the webserver running via `kubectl get pods` and you should see the following logs:

```
kubectl logs fleet-webserver-9bb45dd66-zxnbq
ts=2017-11-16T02:48:38.440578433Z component=service method=ListUsers user=none err=null took=2.350435ms
ts=2017-11-16T02:48:38.441148166Z transport=https address=0.0.0.0:443 msg=listening
```

### Deploying the Load Balancer

Now that the Fleet server is running on our cluster, we have to expose the Fleet webservers to the internet via a load balancer. To create a Kubernetes `Service` of type `LoadBalancer`, run the following:

```
kubectl apply -f ./examples/kubernetes/fleet-service.yml
```

### Configure DNS

Finally, we must configure a DNS address for the external IP address that we now have for the Fleet load balancer. Run the following to show some high-level information about the service:

```
kubectl get services fleet-loadbalancer
```

In this output, you should see an "EXTERNAL-IP" column. If this column says `<pending>`, then give it a few minutes. Sometimes acquiring a public IP address can take a moment.

Once you have the public IP address for the load balancer, create an A record in your DNS server of choice. You should now be able to browse to your Fleet server from the internet!
