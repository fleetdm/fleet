## Fleet on GCP

Required Variables:
```terraform
project_id = "<your project id>"
prefix     = "fleet"
dns_name   = "<the domain you want to host fleet at>" // eg. myfleet.fleetdm.com.
```

### Overview

#### Fleet server
The fleet webserver is running as [Google Cloud Run](https://cloud.google.com/run) containers, this is very similar to how the existing terraform for AWS runs fleet as Fargate compute.
_NOTE: Cloud Run has [limitations](https://cloud.google.com/run/docs/deploying#images) on what container images it will run_. In our deployment we create and deploy the public fleet container image into Artifact Registry.

#### MySQL
We are running MySQL using [Google Cloud SQL](https://cloud.google.com/sql/docs/mysql/introduction) only reachable via [CloudSQLProxy](https://cloud.google.com/sql/docs/mysql/connect-admin-proxy) and from Cloud Run
using [Serverless VPC Access Connector](https://cloud.google.com/sql/docs/mysql/connect-run#private-ip).

#### Redis
We are running Redis using [Google Cloud Memorystore (Redis engine)](https://cloud.google.com/memorystore). This can run in cluster mode, but by default we
are running in standalone mode.

### GCP Managed Certificates

In this example we are using [GCP Managed Certificates](https://cloud.google.com/load-balancing/docs/ssl-certificates/google-managed-certs) to handle TLS and TLS termination at the LoadBalancer.
In order for the certificate to be properly issued, you'll need to update your domain registrar with the nameserver values generated
by the new Zone created in GCP DNS.
