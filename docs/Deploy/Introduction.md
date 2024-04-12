# Advanced deployment

TODO: Page description.

## Introduction
<!-- TODO: video -->

The Fleet application contains two single static binaries which provide web based administration, REST API, and CLI interface to Fleet.

The `fleet` binary contains:
- The Fleet TLS web server (no external webserver is required but it supports a proxy if desired)
- The Fleet web interface
- The Fleet application management [REST API](https://fleetdm.com/docs/using-fleet/rest-api)
- The Fleet osquery API endpoints

The `fleetctl` binary is the CLI interface which allows management of your deployment, scriptable live queries, and easy integration into your existing logging, alerting, reporting, and management infrastructure.

Both binaries are available for download from our [repo](https://github.com/fleetdm/fleet/releases).


## Infrastructure dependencies

Fleet currently has three infrastructure dependencies: MySQL, Redis, and a TLS certificate.

![Fleet's architecture diagram](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/fleet-architecture-diagram.png)

### MySQL

Fleet uses MySQL extensively as its main database. Many cloud providers (such as [AWS](https://aws.amazon.com/rds/mysql/) and [GCP](https://cloud.google.com/sql/)) host reliable MySQL services which you may consider for this purpose. A well-supported MySQL [Docker image](https://hub.docker.com/_/mysql/) also exists if you would rather run MySQL in a container. 
For more information on how to configure the `fleet` binary to use the correct MySQL instance, see the [Configuration](https://fleetdm.com/docs/deploying/configuration) document.

Fleet requires at least MySQL version 5.7, and is tested using the InnoDB storage engine. 

There are many "drop-in replacements" for MySQL available. If you'd like to experiment with some bleeding-edge technology and use Fleet with one of these alternative database servers, we think that's awesome! Please be aware they are not officially supported and that it is very important to set up a dev environment to thoroughly test new releases. 

### Redis

Fleet uses Redis to ingest and queue the results of distributed queries, cache data, etc. Many cloud providers (such as [AWS](https://aws.amazon.com/elasticache/) and [GCP](https://console.cloud.google.com/launcher/details/click-to-deploy-images/redis)) host reliable Redis services which you may consider for this purpose. A well supported Redis [Docker image](https://hub.docker.com/_/redis/) also exists if you would rather run Redis in a container. For more information on how to configure the `fleet` binary to use the correct Redis instance, see the [Configuration](https://fleetdm.com/docs/deploying/configuration) document.

### TLS certificate

In order for osqueryd clients to connect, the connection to Fleet must use TLS. The TLS connection may be terminated by Fleet itself, or by a proxy serving traffic to Fleet.

- The CNAME or one of the Subject Alternate Names (SANs) on the certificate must match the hostname that osquery clients use to connect to the server/proxy.
- If you intend to have your Fleet instance on a subdomain, your certificate can have a wildcard SAN. So `fleet.example.com` should match a SAN of `*.example.com`
- If self-signed certificates are used, the full certificate chain must be provided to osquery via the `--tls_server_certs` flag.
- If Fleet terminates TLS, consider using an ECDSA (rather than RSA) certificate, as RSA certificates have been associated with [performance problems in Fleet due to Go's standard library TLS implementation](https://github.com/fleetdm/fleet/issues/655).

## Reference architectures

You can easily run Fleet on a single VPS that would be capable of supporting hundreds if not thousands of hosts, but
this page details an [opinionated view](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws) of running Fleet in a production environment, as
well as different configuration strategies to enable High Availability (HA).

### Availability components

There are a few strategies that can be used to ensure high availability:
- Database HA
- Traffic load balancing

#### Database HA

Fleet recommends RDS Aurora MySQL when running on AWS. More details about backups/snapshots can be found
[here](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Aurora.Managing.Backups.html). It is also
possible to dynamically scale read replicas to increase performance and [enable database fail-over](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Concepts.AuroraHighAvailability.html).
It is also possible to use [Aurora Global](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-global-database.html) to
span multiple regions for more advanced configurations(_not included in the [reference terraform](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws)_).

In some cases adding a read replica can increase database performance for specific access patterns. In scenarios when automating the API or with `fleetctl`, there can be benefits to read performance.

>Note:Fleet servers need to talk to a writer in the same datacenter. Cross region replication can be used for failover but writes need to be local.

#### Traffic load balancing
Load balancing enables distributing request traffic over many instances of the backend application. Using AWS Application
Load Balancer can also [offload SSL termination](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/create-https-listener.html), freeing Fleet to spend the majority of it's allocated compute dedicated 
to its core functionality. More details about ALB can be found [here](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/introduction.html).

>Note if using [terraform reference architecture](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws#terraform) all configurations can dynamically scale based on load(cpu/memory) and all configurations
assume On-Demand pricing (savings are available through Reserved Instances). Calculations do not take into account NAT gateway charges or other networking related ingress/egress costs.

### Cloud providers

#### AWS

Example configuration breakpoints
##### [Up to 1000 hosts](https://calculator.aws/#/estimate?id=ae7d7ddec64bb979f3f6611d23616b1dff0e8dbd)

| Fleet instances | CPU Units     | RAM |
| --------------- | ------------- | --- |
| 1 Fargate task  | 512 CPU Units | 4GB |

| Dependencies | Version                 | Instance type |
| ------------ | ----------------------- | ------------- |
| Redis        | 6                       | t4g.small     |
| MySQL        | 8.0.mysql_aurora.3.02.0 | db.t3.small   |

##### [Up to 25000 hosts](https://calculator.aws/#/estimate?id=4a3e3168275967d1e79a3d1fcfedc5b17d67a271)

| Fleet instances | CPU Units      | RAM |
| --------------- | -------------- | --- |
| 10 Fargate task | 1024 CPU Units | 4GB |

| Dependencies | Version                 | Instance type |
| ------------ | ----------------------- | ------------- |
| Redis        | 6                       | m6g.large     |
| MySQL        | 8.0.mysql_aurora.3.02.0 | db.r6g.large  |


##### [Up to 150000 hosts](https://calculator.aws/#/estimate?id=1d8fdd63f01e71027e9d898ed05f4a07299a7000)

| Fleet instances | CPU Units      | RAM |
| --------------- | -------------- | --- |
| 20 Fargate task | 1024 CPU Units | 4GB |

| Dependencies | Version                 | Instance type  | Nodes |
| ------------ | ----------------------- | -------------- | ----- |
| Redis        | 6                       | m6g.large      | 3     |
| MySQL        | 8.0.mysql_aurora.3.02.0 | db.r6g.4xlarge | 1     |

##### [Up to 300000 hosts](https://calculator.aws/#/estimate?id=f3da0597a172c6a0a3683023e2700a6df6d42c0b)

| Fleet instances | CPU Units      | RAM |
| --------------- | -------------- | --- |
| 20 Fargate task | 1024 CPU Units | 4GB |

| Dependencies | Version                 | Instance type   | Nodes |
| ------------ | ----------------------- | --------------- | ----- |
| Redis        | 6                       | m6g.large       | 3     |
| MySQL        | 8.0.mysql_aurora.3.02.0 | db.r6g.16xlarge | 2     |

AWS reference architecture can be found [here](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws). This configuration includes:

- VPC
  - Subnets
    - Public & Private
  - ACLs
  - Security Groups
- ECS as the container orchestrator
  - Fargate for underlying compute
  - Task roles via IAM
- RDS Aurora MySQL 8
- Elasticache Redis Engine
- Firehose osquery log destination
  - S3 bucket sync to allow further ingestion/processing
- [Monitoring via Cloudwatch alarms](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws/monitoring)

Some AWS services used in the provider reference architecture are billed as pay-per-use such as Firehose. This means that osquery scheduled query frequency can have
a direct correlation to how much these services cost, something to keep in mind when configuring Fleet in AWS.

##### AWS Terraform CI/CD IAM permissions
The following permissions are the minimum required to apply AWS terraform resources:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:*",
                "cloudwatch:*",
                "s3:*",
                "lambda:*",
                "ecs:*",
                "rds:*",
                "rds-data:*",
                "secretsmanager:*",
                "pi:*",
                "ecr:*",
                "iam:*",
                "aps:*",
                "vpc:*",
                "kms:*",
                "elasticloadbalancing:*",
                "ce:*",
                "cur:*",
                "logs:*",
                "cloudformation:*",
                "ssm:*",
                "sns:*",
                "elasticache:*",
                "application-autoscaling:*",
                "acm:*",
                "route53:*",
                "dynamodb:*",
                "kinesis:*",
                "firehose:*"
            ],
            "Resource": "*"
        }
    ]
}
```

#### GCP

GCP reference architecture can be found in [the Fleet repository](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/gcp). This configuration includes:

- Cloud Run (Fleet backend)
- Cloud SQL MySQL 5.7 (Fleet database)
- Memorystore Redis (Fleet cache & live query orchestrator)

Example configuration breakpoints
##### [Up to 1000 hosts](https://cloud.google.com/products/calculator/#id=59670518-9af4-4044-af4a-cc100a9bed2f)

| Fleet instances | CPU | RAM |
| --------------- | --- | --- |
| 2 Cloud Run     | 1   | 2GB |

| Dependencies | Version               | Instance type |
| ------------ | --------------------- | ------------- |
| Redis        | MemoryStore Redis 6   | M1 Basic      |
| MySQL        | Cloud SQL for MySQL 8 | db-standard-1 |

##### [Up to 25000 hosts](https://cloud.google.com/products/calculator/#id=fadbb96c-967c-4397-9921-743d75b98d42)

| Fleet instances | CPU | RAM |
| --------------- | --- | --- |
| 10 Cloud Run    | 1   | 2GB |

| Dependencies | Version               | Instance type |
| ------------ | --------------------- | ------------- |
| Redis        | MemoryStore Redis 6   | M1 2GB        |
| MySQL        | Cloud SQL for MySQL 8 | db-standard-4 |


##### [Up to 150000 hosts](https://cloud.google.com/products/calculator/#id=baff774c-d294-491f-a9da-dd97bbfa8ef2)

| Fleet instances | CPU   | RAM |
| --------------- | ----- | --- |
| 30 Cloud Run    | 1 CPU | 2GB |

| Dependencies | Version               | Instance type | Nodes |
| ------------ | --------------------- | ------------- | ----- |
| Redis        | MemoryStore Redis 6   | M1 4GB        | 1     |
| MySQL        | Cloud SQL for MySQL 8 | db-highmem-16 | 1     |

#### Azure

Coming soon. Get [commmunity support](https://chat.osquery.io/c/fleet).

#### Render

Using [Render's IAC](https://render.com/docs/infrastructure-as-code) see [the repository](https://github.com/edwardsb/fleet-on-render) for full details.
```yaml
services:
  - name: fleet
    plan: standard
    type: web
    env: docker
    healthCheckPath: /healthz
    envVars:
      - key: FLEET_MYSQL_ADDRESS
        fromService:
          name: fleet-mysql
          type: pserv
          property: hostport
      - key: FLEET_MYSQL_DATABASE
        fromService:
          name: fleet-mysql
          type: pserv
          envVarKey: MYSQL_DATABASE
      - key: FLEET_MYSQL_PASSWORD
        fromService:
          name: fleet-mysql
          type: pserv
          envVarKey: MYSQL_PASSWORD
      - key: FLEET_MYSQL_USERNAME
        fromService:
          name: fleet-mysql
          type: pserv
          envVarKey: MYSQL_USER
      - key: FLEET_REDIS_ADDRESS
        fromService:
          name: fleet-redis
          type: pserv
          property: hostport
      - key: FLEET_SERVER_TLS
        value: false
      - key: PORT
        value: 8080

  - name: fleet-mysql
    type: pserv
    env: docker
    repo: https://github.com/render-examples/mysql
    branch: mysql-5
    disk:
      name: mysql
      mountPath: /var/lib/mysql
      sizeGB: 10
    envVars:
      - key: MYSQL_DATABASE
        value: fleet
      - key: MYSQL_PASSWORD
        generateValue: true
      - key: MYSQL_ROOT_PASSWORD
        generateValue: true
      - key: MYSQL_USER
        value: fleet

  - name: fleet-redis
    type: pserv
    env: docker
    repo: https://github.com/render-examples/redis
    disk:
      name: redis
      mountPath: /var/lib/redis
      sizeGB: 10
```

#### Digital Ocean

Using Digital Ocean's [App Spec](https://docs.digitalocean.com/products/app-platform/concepts/app-spec/) to deploy on the App on the [App Platform](https://docs.digitalocean.com/products/app-platform/)
```yaml
alerts:
- rule: DEPLOYMENT_FAILED
- rule: DOMAIN_FAILED
databases:
- cluster_name: fleet-redis
  engine: REDIS
  name: fleet-redis
  production: true
  version: "6"
- cluster_name: fleet-mysql
  db_name: fleet
  db_user: fleet
  engine: MYSQL
  name: fleet-mysql
  production: true
  version: "8"
domains:
- domain: demo.fleetdm.com
  type: PRIMARY
envs:
- key: FLEET_MYSQL_ADDRESS
  scope: RUN_TIME
  value: ${fleet-mysql.HOSTNAME}:${fleet-mysql.PORT}
- key: FLEET_MYSQL_PASSWORD
  scope: RUN_TIME
  value: ${fleet-mysql.PASSWORD}
- key: FLEET_MYSQL_USERNAME
  scope: RUN_TIME
  value: ${fleet-mysql.USERNAME}
- key: FLEET_MYSQL_DATABASE
  scope: RUN_TIME
  value: ${fleet-mysql.DATABASE}
- key: FLEET_REDIS_ADDRESS
  scope: RUN_TIME
  value: ${fleet-redis.HOSTNAME}:${fleet-redis.PORT}
- key: FLEET_SERVER_TLS
  scope: RUN_AND_BUILD_TIME
  value: "false"
- key: FLEET_REDIS_PASSWORD
  scope: RUN_AND_BUILD_TIME
  value: ${fleet-redis.PASSWORD}
- key: FLEET_REDIS_USE_TLS
  scope: RUN_AND_BUILD_TIME
  value: "true"
jobs:
- envs:
  - key: DATABASE_URL
    scope: RUN_TIME
    value: ${fleet-redis.DATABASE_URL}
  image:
    registry: fleetdm
    registry_type: DOCKER_HUB
    repository: fleet
    tag: latest
  instance_count: 1
  instance_size_slug: basic-xs
  kind: PRE_DEPLOY
  name: fleet-migrate
  run_command: fleet prepare --no-prompt=true db
  source_dir: /
name: fleet
region: nyc
services:
- envs:
  - key: FLEET_VULNERABILITIES_DATABASES_PATH
    scope: RUN_TIME
    value: /home/fleet
  health_check:
    http_path: /healthz
  http_port: 8080
  image:
    registry: fleetdm
    registry_type: DOCKER_HUB
    repository: fleet
    tag: latest
  instance_count: 1
  instance_size_slug: basic-xs
  name: fleet
  routes:
  - path: /
  run_command: fleet serve
  source_dir: /
```

## Advanced deployment guides

Learn how to deploy Fleet on the following platforms:

- [Deploy Fleet on AWS ECS]()
- [Deploy Fleet on AWS with Terraform]()
- [Deploy Fleet on CentOS]()
- [Deploy Fleet on Cloud.gov]()
- [Deploy Fleet on Hetzner Cloud]()
- [Deploy Fleet on Kubernetes]()
- [Deploy Fleet on Render]()

### Community projects

Below are some projects created by Fleet community members. These projects provide additional solutions for deploying Fleet. Please submit a pull request if you'd like your project featured.

- [CptOfEvilMinions/FleetDM-Automation](https://github.com/CptOfEvilMinions/FleetDM-Automation) - Ansible and Docker code to set up Fleet

## Using a proxy

If you are in an enterprise environment where Fleet is behind a proxy and you would like to be able to retrieve vulnerability data for [vulnerability processing](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing), it may be necessary to configure the proxy settings. Fleet automatically uses the `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.

For example, to configure the proxy in a systemd service file:

```systemd
[Service]
Environment="HTTP_PROXY=http(s)://PROXY_URL:PORT/"
Environment="HTTPS_PROXY=http(s)://PROXY_URL:PORT/"
Environment="NO_PROXY=localhost,127.0.0.1,::1"
```

After modifying the configuration you will need to reload and restart the Fleet service, as explained above.

## Public IPs of devices

Fleet attempts to deduce the public IP of devices from well-known HTTP headers received on requests made by the osquery agent.

The HTTP request headers are checked in the following order:
1. If `True-Client-IP` header is set, then Fleet will extract its value.
2. If `X-Real-IP` header is set, then Fleet will extract its value.
3. If `X-Forwarded-For` header is set, then Fleet will extract the first comma-separated value.
4. If none of the above headers are present in the HTTP request then Fleet will attempt to use the remote address of the TCP connection (note that on deployments with ingress proxies the remote address seen by Fleet is the IP of the ingress proxy).

## Single sign-on (SSO)

Learn how to configure single sign-on (SSO) and just-in-time (JIT) user provisioning.

#### In this section

- [Overview](#overview)
- [Indentity provider (IDP) configuration](#indentity-provider-idp-configuration)
- [Fleet SSO configuration](#fleet-sso-configuration)
- [Creating SSO users in Fleet](#creating-sso-users-in-fleet)
- [Enabling SSO for existing users in Fleet](#enabling-sso-for-existing-users-in-fleet)
- [Just-in-time (JIT) user provisioning](#just-in-time-jit-user-provisioning)


### Overview

Fleet supports SAML single sign-on capability.

Fleet supports both SP-initiated SAML login and IDP-initiated login. However, IDP-initiated login must be enabled in the web interface's SAML single sign-on options.

Fleet supports the SAML Web Browser SSO Profile using the HTTP Redirect Binding.

> Note: The email used in the SAML Assertion must match a user that already exists in Fleet unless you enable [JIT provisioning](#just-in-time-jit-user-provisioning).

### Identity provider (IDP) configuration

Setting up the service provider (Fleet) with an identity provider generally requires the following information:

- _Assertion Consumer Service_ - This is the call-back URL that the identity provider
  will use to send security assertions to Fleet. In Okta, this field is called _single sign-on URL_. On Google, it is "ACS URL." The value you supply will be a fully qualified URL consisting of your Fleet web address and the call-back path `/api/v1/fleet/sso/callback`. For example, if your Fleet web address is https://fleet.example.com, then the value you would use in the identity provider configuration would be:
  ```text
  https://fleet.example.com/api/v1/fleet/sso/callback
  ```

- _Entity ID_ - This value is an identifier that you choose. It identifies your Fleet instance as the service provider that issues authorization requests. The value must match the Entity ID that you define in the Fleet SSO configuration.

- _Name ID Format_ - The value should be `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`. This may be shortened in the IDP setup to something like `email` or `EmailAddress`.

- _Subject Type (Application username in Okta)_ - `email`.

After supplying the above information, the IDP will generate an issuer URI and metadata that will be used to configure Fleet as a service provider.

### Fleet SSO configuration

A Fleet user must be assigned the Admin role to configure Fleet for SSO. In Fleet, SSO configuration settings are located in **Settings > Organization settings > SAML single sign-on options**.

If your IDP supports dynamic configuration, like Okta, you only need to provide an _identity provider name_ and _entity ID_, then paste a link in the metadata URL field. Make sure you create the SSO application within your IDP before configuring it in Fleet.

Otherwise, the following values are required:

- _Identity provider name_ - A human-readable name of the IDP. This is rendered on the login page.

- _Entity ID_ - A URI that identifies your Fleet instance as the issuer of authorization
  requests (e.g., `fleet.example.com`). This must match the _Entity ID_ configured with the IDP.

- _Metadata URL_ - Obtain this value from the IDP and is used by Fleet to
  issue authorization requests to the IDP.

- _Metadata_ - If the IDP does not provide a metadata URL, the metadata must
  be obtained from the IDP and entered. Note that the metadata URL is preferred if
  the IDP provides metadata in both forms.

#### Example Fleet SSO configuration

![Example SSO Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/sso-setup.png)

### Creating SSO users in Fleet

When an admin creates a new user in Fleet, they may select the `Enable single sign on` option. The
SSO-enabled users will not be able to sign in with a regular user ID and password.

It is strongly recommended that at least one admin user is set up to use the traditional password-based login so that there is a fallback method for logging into Fleet in the event of SSO
configuration problems.

> Individual users must also be set up on the IDP before signing in to Fleet.

### Enabling SSO for existing users in Fleet
As an admin, you can enable SSO for existing users in Fleet. To do this, go to the Settings page,
then click on the Users tab. Locate the user you want to enable SSO for, and in the Actions dropdown
menu for that user, click on "Edit." In the dialogue that opens, check the box labeled "Enable
single sign-on," then click "Save." If you are unable to check that box, you must first [configure
and enable SSO for the organization](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso).

### Just-in-time (JIT) user provisioning

`Applies only to Fleet Premium`

When JIT user provisioning is turned on, Fleet will automatically create an account when a user logs in for the first time with the configured SSO. This removes the need to create individual user accounts for a large organization.

The new account's email and full name are copied from the user data in the SSO response.
By default, accounts created via JIT provisioning are assigned the [Global Observer role](https://fleetdm.com/docs/using-fleet/permissions).
To assign different roles for accounts created via JIT provisioning see [Customization of user roles](#customization-of-user-roles) below.

To enable this option, go to **Settings > Organization settings > Single sign-on options** and check "_Create user and sync permissions on login_" or [adjust your config](#sso-settings-enable-jit-provisioning).

For this to work correctly make sure that:

- Your IDP is configured to send the user email as the Name ID (instructions for configuring different providers are detailed below)
- Your IDP sends the full name of the user as an attribute with any of the following names (if this value is not provided Fleet will fallback to the user email)
  - `name`
  - `displayname`
  - `cn`
  - `urn:oid:2.5.4.3`
  - `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name`

#### Customization of user roles

> **Note:** This feature requires setting `sso_settings.enable_jit_provisioning` to `true`.

Users created via JIT provisioning can be assigned Fleet roles using SAML custom attributes that are sent by the IdP in `SAMLResponse`s during login.
Fleet will attempt to parse SAML custom attributes with the following format:
- `FLEET_JIT_USER_ROLE_GLOBAL`: Specifies the global role to use when creating the user.
- `FLEET_JIT_USER_ROLE_TEAM_<TEAM_ID>`: Specifies team role for team with ID `<TEAM_ID>` to use when creating the user.

Currently supported values for the above attributes are: `admin`, `maintainer`, `observer`, `observer_plus` and `null`.
A role attribute with value `null` will be ignored by Fleet. (This is to support limitations on some IdPs which do not allow you to choose what keys are sent to Fleet when creating a new user.)
SAML supports multi-valued attributes, Fleet will always use the last value.

NOTE: Setting both `FLEET_JIT_USER_ROLE_GLOBAL` and `FLEET_JIT_USER_ROLE_TEAM_<TEAM_ID>` will cause an error during login as Fleet users cannot be Global users and belong to teams.

Following is the behavior that will take place on every SSO login:

If the account does not exist then:
  - If the `SAMLResponse` has any role attributes then those will be used to set the account roles.
  - If the `SAMLResponse` does not have any role attributes set, then Fleet will default to use the `Global Observer` role.

If the account already exists:
  - If the `SAMLResponse` has any role attributes then those will be used to update the account roles.
  - If the `SAMLResponse` does not have any role attributes set, no role change is attempted.

Here's a `SAMLResponse` sample to set the role of SSO users to Global `admin`:
```xml
[...]
<saml2:Assertion ID="id16311976805446352575023709" IssueInstant="2023-02-27T17:41:53.505Z" Version="2.0" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <saml2:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">http://www.okta.com/exk8glknbnr9Lpdkl5d7</saml2:Issuer>
  [...]
  <saml2:Subject xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">bar@foo.example.com</saml2:NameID>
    <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml2:SubjectConfirmationData InResponseTo="id1Juy6Mx2IHYxLwsi" NotOnOrAfter="2023-02-27T17:46:53.506Z" Recipient="https://foo.example.com/api/v1/fleet/sso/callback"/>
    </saml2:SubjectConfirmation>
  </saml2:Subject>
  [...]
  <saml2:AttributeStatement xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:Attribute Name="FLEET_JIT_USER_ROLE_GLOBAL" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
      <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">admin</saml2:AttributeValue>
    </saml2:Attribute>
  </saml2:AttributeStatement>
</saml2:Assertion>
[...]
```

Here's a `SAMLResponse` sample to set the role of SSO users to `observer` in team with ID `1` and `maintainer` in team with ID `2`:
```xml
[...]
<saml2:Assertion ID="id16311976805446352575023709" IssueInstant="2023-02-27T17:41:53.505Z" Version="2.0" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <saml2:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity" xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">http://www.okta.com/exk8glknbnr9Lpdkl5d7</saml2:Issuer>
  [...]
  <saml2:Subject xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">bar@foo.example.com</saml2:NameID>
    <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
      <saml2:SubjectConfirmationData InResponseTo="id1Juy6Mx2IHYxLwsi" NotOnOrAfter="2023-02-27T17:46:53.506Z" Recipient="https://foo.example.com/api/v1/fleet/sso/callback"/>
    </saml2:SubjectConfirmation>
  </saml2:Subject>
  [...]
  <saml2:AttributeStatement xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">
    <saml2:Attribute Name="FLEET_JIT_USER_ROLE_TEAM_1" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
      <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">observer</saml2:AttributeValue>
    </saml2:Attribute>
    <saml2:Attribute Name="FLEET_JIT_USER_ROLE_TEAM_2" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified">
      <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string">maintainer</saml2:AttributeValue>
    </saml2:Attribute>
  </saml2:AttributeStatement>
</saml2:Assertion>
[...]
```

Each IdP will have its own way of setting these SAML custom attributes, here are instructions for how to set it for Okta: https://support.okta.com/help/s/article/How-to-define-and-configure-a-custom-SAML-attribute-statement?language=en_US.

#### Okta IDP configuration

![Example Okta IDP Configuration](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-idp-setup.png)

Once configured, you will need to retrieve the Issuer URI from the `View Setup Instructions` and metadata URL from the `Identity Provider metadata` link within the application `Sign on` settings. See below for where to find them:

![Where to find SSO links for Fleet](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/okta-retrieve-links.png)

> The Provider Sign-on URL within the `View Setup Instructions` has a similar format as the Provider SAML Metadata URL, but this link provides a redirect to _sign into_ the application, not the metadata necessary for dynamic configuration.

> The names of the items required to configure an identity provider may vary from provider to provider and may not conform to the SAML spec.

#### Google Workspace IDP Configuration

Follow these steps to configure Fleet SSO with Google Workspace. This will require administrator permissions in Google Workspace.

1. Navigate to the [Web and Mobile Apps](https://admin.google.com/ac/apps/unified) section of the Google Workspace dashboard. Click _Add App -> Add custom SAML app_.

  ![The Google Workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-1.png)

2. Enter `Fleet` for the _App name_ and click _Continue_.

  ![Adding a new app to Google workspace admin dashboard](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-2.png)

3. Click _Download Metadata_, saving the metadata to your computer. Click _Continue_.

  ![Download metadata](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-3.png)

4. In Fleet, navigate to the _Organization Settings_ page. Configure the _SAML single sign-on options_ section.

  - Check the _Enable single sign-on_ checkbox.
  - For _Identity provider name_, use `Google`.
  - For _Entity ID_, use a unique identifier such as `fleet.example.com`. Note that Google seems to error when the provided ID includes `https://`.
  - For _Metadata_, paste the contents of the downloaded metadata XML from step three.
  - All other fields can be left blank.

  Click _Update settings_ at the bottom of the page.

  ![Fleet's SAML single sign on options page](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-4.png)

5. In Google Workspace, configure the _Service provider details_.

  - For _ACS URL_, use `https://<your_fleet_url>/api/v1/fleet/sso/callback` (e.g., `https://fleet.example.com/api/v1/fleet/sso/callback`).
  - For Entity ID, use **the same unique identifier from step four** (e.g., `fleet.example.com`).
  - For _Name ID format_, choose `EMAIL`.
  - For _Name ID_, choose `Basic Information > Primary email`.
  - All other fields can be left blank.

  Click _Continue_ at the bottom of the page.

  ![Configuring the service provider details in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-5.png)

6. Click _Finish_.

  ![Finish configuring the new SAML app in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-6.png)

7. Click the down arrow on the _User access_ section of the app details page.

  ![The new SAML app's details page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-7.png)

8. Check _ON for everyone_. Click _Save_.

  ![The new SAML app's service status page in Google Workspace](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/google-sso-configuration-step-8.png)

9. Enable SSO for a test user and try logging in. Note that Google sometimes takes a long time to propagate the SSO configuration, and it can help to try logging in to Fleet with an Incognito/Private window in the browser.

## Systemd


### Run with systemd

Once you've verified that you can run Fleet in your shell, you'll likely want to keep Fleet running in the background and after the server reboots. To do that we recommend using [systemd](https://coreos.com/os/docs/latest/getting-started-with-systemd.html).

Below is a sample unit file, assuming a `fleet` user exists on the system. Any user with sufficient
permissions to execute the binary, open the configuration files, and write the log files can be
used. It is also possible to run as `root`, though as with any other web server it is discouraged
to run Fleet as `root`.

```systemd

[Unit]
Description=Fleet
After=network.target

[Service]
User=fleet
Group=fleet
LimitNOFILE=8192
ExecStart=/usr/local/bin/fleet serve \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=root \
  --mysql_password=toor \
  --redis_address=127.0.0.1:6379 \
  --server_cert=/tmp/server.cert \
  --server_key=/tmp/server.key \
  --logging_json

[Install]
WantedBy=multi-user.target
```

Once you created the file, you need to move it to `/etc/systemd/system/fleet.service` and start the service.

```sh
sudo mv fleet.service /etc/systemd/system/fleet.service
sudo systemctl start fleet.service
sudo systemctl status fleet.service

sudo journalctl -u fleet.service -f
```

### Making changes

Sometimes you'll need to update the systemd unit file defining the service. To do that, first open /etc/systemd/system/fleet.service in a text editor, and make your modifications.

Then, run

```sh
sudo systemctl daemon-reload
sudo systemctl restart fleet.service
```

## Monitoring Fleet

#### In this section

- [Health checks](#health-checks)
- [Metrics](#metrics)
- [Fleet server performance](#fleet-server-performance)

### Health checks

Fleet exposes a basic health check at the `/healthz` endpoint. This is the interface to use for simple monitoring and load-balancer health checks.

The `/healthz` endpoint will return an `HTTP 200` status if the server is running and has healthy connections to MySQL and Redis. If there are any problems, the endpoint will return an `HTTP 500` status. Details about failing checks are logged in the Fleet server logs.

Individual checks can be run by providing the `check` URL parameter (e.x., `/healthz?check=mysql` or `/healthz?check=redis`).
### Metrics

Fleet exposes server metrics in a format compatible with [Prometheus](https://prometheus.io/). A simple example Prometheus configuration is available in [tools/app/prometheus.yml](https://github.com/fleetdm/fleet/blob/194ad5963b0d55bdf976aa93f3de6cabd590c97a/tools/app/prometheus.yml).

Prometheus can be configured to use a wide range of service discovery mechanisms within AWS, GCP, Azure, Kubernetes, and more. See the Prometheus [configuration documentation](https://prometheus.io/docs/prometheus/latest/configuration/configuration/) for more information.

#### Alerting

##### Prometheus

Prometheus has built-in support for alerting through [Alertmanager](https://prometheus.io/docs/alerting/latest/overview/).

Consider building alerts for

- Changes from expected levels of host enrollment
- Increased latency on HTTP endpoints
- Increased error levels on HTTP endpoints

```
TODO (Seeking Contributors)
Add example alerting configurations
```

##### Cloudwatch Alarms

Cloudwatch Alarms can be configured to support a wide variety of metrics and anomaly detection mechanisms. There are some example alarms
in the terraform reference architecture (see `monitoring.tf`).

* [Monitoring RDS (MySQL)](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/monitoring-cloudwatch.html)
* [ElastiCache for Redis](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.WhichShouldIMonitor.html)
* [Monitoring ECS](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/cloudwatch-metrics.html)
* Reference alarms include evaluating healthy targets & response times. We also use target-tracking alarms to manage auto-scaling.

#### Graphing

Prometheus provides basic graphing capabilities, and integrates tightly with [Grafana](https://prometheus.io/docs/visualization/grafana/) for sophisticated visualizations.

### Fleet server performance

Fleet is designed to scale to hundreds of thousands of online hosts. The Fleet server scales horizontally to support higher load.

#### Horizontal scaling

Scaling Fleet horizontally is as simple as running more Fleet server processes connected to the same MySQL and Redis backing stores. Typically, operators front Fleet server nodes with a load balancer that will distribute requests to the servers. All APIs in Fleet are designed to work in this arrangement by simply configuring clients to connect to the load balancer.

#### Availability

The Fleet/osquery system is resilient to loss of availability. Osquery agents will continue executing the existing configuration and buffering result logs during downtime due to lack of network connectivity, server maintenance, or any other reason. Buffering in osquery can be configured with the `--buffered_log_max` flag.

Note that short downtimes are expected during [Fleet server upgrades](https://fleetdm.com/docs/deploying/upgrading-fleet) that require database migrations.

#### Debugging performance issues

##### MySQL and Redis

If performance issues are encountered with the MySQL and Redis servers, use the extensive resources available online to optimize and understand these problems. Please [file an issue](https://github.com/fleetdm/fleet/issues/new/choose) with details about the problem so that Fleet developers can work to fix them.

##### Fleet server

For performance issues in the Fleet server process, please [file an issue](https://github.com/fleetdm/fleet/issues/new/choose) with details about the scenario, and attach a debug archive. Debug archives can also be submitted confidentially through other support channels.

###### Generate debug archive (Fleet 3.4.0+)

Use the `fleetctl debug archive` command to generate an archive of Fleet's full suite of debug profiles. See the [fleetctl setup guide](https://fleetdm.com/docs/using-fleet/fleetctl-cli) for details on configuring `fleetctl`.

The generated `.tar.gz` archive will be available in the current directory.

###### Targeting individual servers

In most configurations, the `fleetctl` client is configured to make requests to a load balancer that will proxy the requests to each server instance. This can be problematic when trying to debug a performance issue on a specific server. To target an individual server, create a new `fleetctl` context that uses the direct address of the server.

For example:

```sh
fleetctl config set --context server-a --address https://server-a:8080
fleetctl login --context server-a
fleetctl debug archive --context server-a
```

###### Confidential information

The `fleetctl debug archive` command retrieves information generated by Go's [`net/http/pprof`](https://golang.org/pkg/net/http/pprof/) package. In most scenarios this should not include sensitive information, however it does include command line arguments to the Fleet server. If the Fleet server receives sensitive credentials via CLI argument (not environment variables or config file), this information should be scrubbed from the archive in the `cmdline` file.

<meta name="pageOrderInSection" value="100">
<meta name="description" value="Learn about Fleet's architecture and infrastructure dependencies.">
