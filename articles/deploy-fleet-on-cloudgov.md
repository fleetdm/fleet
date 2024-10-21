# Deploy Fleet on Cloud.gov (Cloud Foundry)

![Deploy Fleet on Cloud.gov](../website/assets/images/articles/deploy-fleet-on-cloudgov-800x450@2x.png)

Cloud.gov is a [FEDRAMP moderate Platform-as-a-Service
(PaaS)](https://marketplace.fedramp.gov/#!/product/18f-cloudgov?sort=productName). This repository
includes a sample [manifest.yml.example](https://github.com/fleetdm/fleet/blob/main/manifest.yml.cloudgov.example) file ready to be pushed and run the
latest version of fleetdm as a container. Make a copy of the example file and update the key
values as appropriate.

1. `cp manifest.yml.cloudgov.example manifest.yml`

2. Setup a cloud.gov account - https://cloud.gov/docs/getting-started/accounts/ 

3. Install the cf-cli - https://cloud.gov/docs/getting-started/setup/

4. From your local fleetdm source directory.

```sh
git clone https://github.com/fleetdm/fleet
cd fleet
cf login -a api.fr.cloud.gov  --sso

# Follow the link to copy the Temporary Authentication Code when prompted
```

5. Setup a demo application space

```sh
cf target -o sandbox-gsa create-space fleetdm-rename
```

6. Create database(s)

```sh
# Update manifest.yml file to rename application and database key names to match commands below.

cf marketplace
cf create-service aws-rds medium-mysql fleetdm-mysql
cf create-service aws-elasticache-redis redis-dev fleetdm-redis
cf create-service-key fleetdm-db-rename fleetdm-db-test-key
cf push
```

**You will be returned the URL for your new test instance to navigate to.**

> Note: This is only for demonstration purposes, in order to run a production level federal/FISMA system. You will need to contact the [cloud.gov program](https://cloud.gov) and consult your organization's security team (for risk assessment and an Authority to Operate).


## Using jq to map service credentials

Cloud foundry injects an environmental variable [`$VCAP_SERVICES`](https://docs.cloudfoundry.org/devguide/deploy-apps/environment-variable.html#VCAP-SERVICES) which is available within the
container. fleetdm uses jq to map service injected credentials to the [standard fleetdm environment
variables](https://fleetdm.com/docs/deploying/configuration#using-only-environment-variables). 

```json
{
    "aws-rds": [
        {
            "label": "aws-rds",
            "provider": null,
            "plan": "medium-mysql",
            "name": "fleetdm-db",
            "tags": [
                "database",
                "RDS"
            ],
            "instance_guid": "guid",
            "instance_name": "fleetdm-db",
            "binding_guid": "guid",
            "binding_name": null,
            "credentials": {
                "db_name": "db_name",
                "host": "host",
                "name": "name",
                "password": "password",
                "port": "3306",
                "uri": "mysql://username:password@hostname:port/db_name",
                "username": "username"
            },
            "syslog_drain_url": null,
            "volume_mounts": []
        }
    ],
    "aws-elasticache-redis": [
        {
            "label": "aws-elasticache-redis",
            "provider": null,
            "plan": "redis-dev",
            "name": "fleetdm-redis",
            "tags": [
                "redis",
                "Elasticache",
                "AWS"
            ],
            "instance_guid": "guid",
            "instance_name": "fleetdm-redis",
            "binding_guid": "guid",
            "binding_name": null,
            "credentials": {
                "current_redis_engine_version": "version",
                "host": "host",
                "hostname": "hostname",
                "password": "password",
                "port": "port",
                "uri": "redis://:address:port"
            },
            "syslog_drain_url": null,
            "volume_mounts": []
        }
    ]
}
```

<meta name="articleTitle" value="Deploy Fleet on Cloud.gov">
<meta name="authorGitHubUsername" value="JJediny">
<meta name="authorFullName" value="John Jediny">
<meta name="publishedOn" value="2022-09-08">
<meta name="category" value="guides">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-fleet-on-cloudgov-800x450@2x.png">
<meta name="description" value="Information for deploying Fleet on Cloud.gov.">
