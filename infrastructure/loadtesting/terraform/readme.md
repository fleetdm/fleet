## Terraform for Loadtesting Environment

The interface into this code is designed to be minimal.
If you require changes beyond whats described here, contact #g-infra.

### Deployment sizing

When loadtesting, it is important to size your load test for the number of hosts you plan to use.  Please see https://fleetdm.com/docs/deploy/reference-architectures for some examples.

These are set via [variables](https://github.com/fleetdm/fleet/blob/main/infrastructure/loadtesting/terraform/variables.tf) and should be applied to every terraform operation.  Below is an example for a modest (~5k) number of hosts:

```sh
# When first applying.  Assuming tag exists
terraform apply -var tag=hosts-5k-test -var fleet_containers=5 -var db_instance_type=db.t4g.medium -var redis_instance_type=cache.t4g.small

# When adding loadtest containers. 
terraform apply -var tag=hosts-5k-test -var fleet_containers=5 -var db_instance_type=db.t4g.medium -var redis_instance_type=cache.t4g.small -var -var loadtest_containers=10 
```

### Deploying your code to the loadtesting environment

> IMPORTANT:
> - We advice to use a separate clone of the https://github.com/fleetdm/fleet repository because `terraform` operations are lengthy. Terraform uses the local files as the configuration files.
> - When performing a load test you target a specific branch and not `main` (referenced below as `$BRANCH_NAME`). The `main` branch changes often and it might trigger rebuilts of the images. The cloned repository that you will use to run the terraform operations doesn't need to be in `$BRANCH_NAME`, such `$BRANCH_NAME` is the Fleet version that will be deployed to the load test environment.
> - These scripts were tested with terraform 1.10.4.

1. Push your `$BRANCH_NAME` branch to https://github.com/fleetdm/fleet and trigger a manual run of the [Docker publish](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-snapshot-fleet.yaml) workflow (make sure to select the branch).
1. arm64 (M1/M2/etc) Mac Only: run `helpers/setup-darwin_arm64.sh` to build terraform plugins that lack arm64 builds in the registry.  Alternatively, you can use the amd64 terraform binary, which works with Rosetta 2.
1. Log into AWS SSO on `loadtesting` via `aws sso login`. (If you have multiple profiles, export the `AWS_PROFILE` variable.) For configuration, see `infrastructure/sso` folder's readme in the `confidential` private repo.
1. Initialize your terraform environment with `terraform init`.
1. Select a workspace for your test: `terraform workspace new WORKSPACE-NAME; terraform workspace select WORKSPACE-NAME`. Ensure your `WORKSPACE-NAME` is less than or equal to 17 characters and contains only lowercase alphanumeric characters and hyphens, as it is used to generate names for AWS resources.
1. Apply terraform with your branch name with `terraform apply -var tag=BRANCH_NAME` and type `yes` to approve execution of the plan. This takes a while to complete (many minutes, > ~30m). Note that for a few minutes after `terraform apply`, the Fleet instances may be failing to start with a permission issue (to read a database secret), but this should resolve automatically after a bit and ECS will begin to start the Fleet instances, but they may still fail due to missing database migrations (this will show up in the instances' logs). At this point you can move on to the next step.
1. Run database migrations (see [Running migrations](#running-migrations)). You will get 500 errors and your containers will not run if you do not do this. After running this step, you might need to wait a few minutes until the environment is up and running.
1. Perform your tests (see [Running a loadtest](#running-a-loadtest)). Your deployment will be available at `https://WORKSPACE-NAME.loadtest.fleetdm.com`. Reach out to the infrastructure team to get the credentials to log in.
1. For instructions on how to deploy new code changes to Fleet to the environment, see [Deploying code changes to Fleet](#deploying-code-changes-to-fleet). This is useful to test performance improvements without having to set up a new loadtest environment.
1. When you're done, clean up the environment with `terraform destroy` (it will prompt for the branch name). If A destroy fails, see [ECR Cleanup Troubleshooting](#ecr-cleanup-troubleshooting) for the most common reason.

### Running migrations

After applying terraform with the commands above and before performing your tests, run the following command:
`aws ecs run-task --region us-east-2 --cluster fleet-"$(terraform workspace show)"-backend --task-definition fleet-"$(terraform workspace show)"-migrate:"$(terraform output -raw fleet_migration_revision)" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="$(terraform output -raw fleet_migration_subnets)",securityGroups="$(terraform output -raw fleet_migration_security_groups)"}"`

### MDM

If you need to run a load test with MDM enabled and configured you will need to set MDM certificates, keys and tokens to the Fleet configuration.

1. Place the files in a known location:
```sh
/Users/foobar/mdm/fleet-mdm-apple-scep.crt
/Users/foobar/mdm/fleet-mdm-apple-scep.key

/Users/foobar/mdm/mdmcert.download.push.pem
/Users/foobar/mdm/mdmcert.download.push.key

/Users/foobar/mdm/downloadtoken.p7m

/Users/foobar/mdm/fleet-apple-mdm-bm-public-key.crt
/Users/foobar/mdm/fleet-apple-mdm-bm-private.key
```

2. Then set the `fleet_config` terraform var the following way (make sure to add any extra configuration you need to this JSON):
```sh
export TF_VAR_fleet_config='{"FLEET_DEV_MDM_APPLE_DISABLE_PUSH":"1","FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY":"1","FLEET_MDM_APPLE_SCEP_CHALLENGE":"foobar","FLEET_MDM_APPLE_SCEP_CERT_BYTES":"'$(cat /Users/foobar/mdm/fleet-mdm-apple-scep.crt | gsed -z 's/\n/\\n/g')'","FLEET_MDM_APPLE_SCEP_KEY_BYTES":"'$(cat /Users/foobar/mdm/fleet-mdm-apple-scep.key | gsed -z 's/\n/\\n/g')'","FLEET_MDM_APPLE_APNS_CERT_BYTES":"'$(cat /Users/foobar/mdm/mdmcert.download.push.pem | gsed -z 's/\n/\\n/g')'","FLEET_MDM_APPLE_APNS_KEY_BYTES":"'$(cat /Users/foobar/mdm/mdmcert.download.push.key | gsed -z 's/\n/\\n/g')'","FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES":"'$(cat /Users/foobar/mdm/downloadtoken.p7m | gsed -z 's/\n/\\n/g' | gsed 's/"smime\.p7m"/\\"smime.p7m\\"/g' | tr -d '\r\n')'","FLEET_MDM_APPLE_BM_CERT_BYTES":"'$(cat /Users/foobar/mdm/fleet-apple-mdm-bm-public-key.crt | gsed -z 's/\n/\\n/g')'","FLEET_MDM_APPLE_BM_KEY_BYTES":"'$(cat /Users/foobar/mdm/fleet-apple-mdm-bm-private.key | gsed -z 's/\n/\\n/g')'"}'
```

- The above is needed because the newline characters in the certificate/key/token files.
- The value set in `FLEET_MDM_APPLE_SCEP_CHALLENGE` must match whatever you set in `osquery-perf`'s `mdm_scep_challenge` argument. 
- The above `export TF_VAR_fleet_config=...` command was tested on `bash`. It did not work in `zsh`.
- Note that we are also setting `FLEET_DEV_MDM_APPLE_DISABLE_PUSH=1`. We don't want to generate push notifications against fake UUIDs (otherwise it may cause Apple to rate limit due to invalid requests).
- Note that we are also setting `FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY=1` to skip verification of Apple certificates for OTA enrollments.
This has an impact on real devices because they will not be notified of any command to execute (it may take a reboot for them to reach out to Fleet for more commands).

3. Add the following `osquery-perf` arguments to [loadtesting.tf](./loadtesting.tf)
- `-mdm_prob 1.0`
- `-mdm_scep_challenge` set to the same value as `FLEET_MDM_APPLE_SCEP_CHALLENGE` above.

### Running a loadtest

We run simulated hosts in containers of 500 at a time. Once the infrastructure is running, you can run the following command:

`terraform apply -var tag=BRANCH_NAME -var loadtest_containers=8`

With the variable `loadtest_containers` you can specify how many containers of 500 hosts you want to start. In the example above, it will run 4000. If the `fleet` instances need special configuration, you can pass them as environment variables to the `fleet_config` terraform variable, which is a map, using the following syntax (note the use of single quotes around the whole `fleet_config` variable assignment, and the use of double quotes inside its map value):

`terraform apply -var tag=BRANCH_NAME -var loadtest_containers=8 -var='fleet_config={"FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING":"host_last_seen=true","FLEET_OSQUERY_ASYNC_HOST_COLLECT_INTERVAL":"host_last_seen=10s"}'`

### Monitoring the infrastructure

There are a few main places of interest to monitor the load and resource usage:

* The Application Performance Monitoring (APM) dashboard: access it on your Fleet load-testing URL on port `:5601` and path `/app/apm`, e.g. `https://loadtest.fleetdm.com:5601/app/apm`.  Note to do this without the VPN you will need to add your public IP Address to the load balancer for TCP Port 5601.  At the time of this writing, [this](https://us-east-2.console.aws.amazon.com/vpc/home?region=us-east-2#SecurityGroup:groupId=sg-0e67d910a662720f8) will take you directly to the security group for the load balancer if logged into the Load Testing account.
* The APM dashboard can also be accessed via private IP over the VPN.  Use the following one-liner to get the URL: `aws ec2 describe-instances --region=us-east-2 | jq -r '.Reservations[].Instances[] | select(.State.Name == "running") | select(.Tags[] | select(.Key == "ansible_playbook_file") | .Value == "elasticsearch.yml") | "http://" + .PrivateIpAddress + ":5601/app/apm"'`.  This connects directly to the EC2 instance and doesn't use the load balancer.
* To monitor mysql database load, go to AWS RDS, select "Performance Insights" and the database instance to monitor (you may want to turn off auto-refresh).
* To monitor Redis load, go to Amazon ElastiCache, select the redis cluster to monitor, and go to "Metrics".

### Deploying code changes to Fleet

You can deploy new code changes to an environment the following way:

1. Push the code changes to the `BRANCH_NAME`, trigger a manual run of the [Docker publish](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-snapshot-fleet.yaml) workflow (make sure to select the branch) and wait for it to complete.
2. Find the docker image IDs corresponding to your branch:
```sh
docker images | grep 'BRANCH_NAME' | awk '{print $3}'
```
3. Remove such image IDs with `docker rmi $IMAGE_ID`.
4. Run the following to trigger a re-deploy of the Fleet instances with the new Fleet docker image:
```sh
# - You must set `loadtest_containers` to the current count (otherwise it will bring the currently running simulated hosts down)
# - If we don't specify the `-target`s then it will bring the loadtest containers down and re-deploy them with the new image, we don't want that because
# you will end up with twice the hosts enrolled (half online, half offline).
terraform apply -var tag=BRANCH_NAME -var loadtest_containers=XXX -target=aws_ecs_service.fleet -target=aws_ecs_task_definition.backend -target=aws_ecs_task_definition.migration -target=aws_s3_bucket_acl.osquery-results -target=aws_s3_bucket_acl.osquery-status -target=docker_registry_image.fleet
```

### Deploying code changes to osquery-perf

Following are the steps to deploy new code changes to osquery-perf (known as `loadtest` image in ECS) on a running loadtest environment.

> osquery-perf simulator in ECS doesn't keep state so you cannot change existing hosts to use new osquery-perf code.
> The following is to add new hosts with new/modified osquery-perf code. (This happens if during a load test
> the developer realizes there's bug in osquery-perf or if it's not simulating osquery properly.)

> You must push your code changes to the `$BRANCH_NAME`.

1. Bring all `loadtest` containers to `0` by running terraform apply with `loadtest_containers=0`.
1. Delete all existing hosts (by selecting all on the UI).
1. Delete all your local `loadtest` images, the image tags are of the form: `loadtest-$BRANCH_NAME-$TAG` (these are the `loadtest` images pushed to ECR).  (Use `docker image list` to get their `IMAGE ID` and then run `docker rmi -f $ID`.)
1. Delete local images of the form `REPOSITORY=<none>` and `TAG=<none>` that were built recently (these are the builder images). (Use `docker image list` to get their `IMAGE ID` and then run `docker rmi -f $ID`.)
1. Log in to Amazon ECR (Elastic Container Registry) and delete the corresponding `loadtest` image.
1. By executing the `terraform apply` with `-loadtest_containers=N` it will trigger a rebuild of the `loadtest` image.

### Troubleshooting

#### Using a release tag instead of a branch

Since the tag name on Dockerhub doesn't match the tag name on GitHub, this presents a special use case when wanting to deploy a release tag.  In this case, you can use the optional `-var git_branch` in order to specify the separate tag.  For example, you would use the following to deploy a loadtest of version 4.28.0:

`terraform apply -var tag=v4.28.0 -var git_branch=fleet-v4.28.0 -var loadtest_containers=8`

#### General Troubleshooting

If terraform fails for some reason, you can make it output extra information to `stderr` by setting the `TF_LOG` environment variable to "DEBUG" or "TRACE", e.g.:

`TF_LOG=DEBUG terraform apply ...`

See https://www.terraform.io/internals/debugging for more details.

#### ECR Cleanup Troubleshooting

In a few instances, it is possible for an ECR repository to still have images left, preventing a full `terraform destroy` of a Loadtesting instance.  Use the following one-liner to clean these up before re-running `terraform destroy`:

`REPOSITORY_NAME=fleet-$(terraform workspace show); aws ecr list-images --repository-name ${REPOSITORY_NAME} --query 'imageIds[*]' --output text | while read digest tag; do aws ecr batch-delete-image --repository-name ${REPOSITORY_NAME} --image-ids imageDigest=${digest}; done`

### Errors with macOS Docker Desktop

If you are getting the following error when running `terraform apply`:
```sh
│ Error: Error pinging Docker server: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
│
│   with provider["registry.terraform.io/kreuzwerker/docker"],
│   on init.tf line 45, in provider "docker":
│   45: provider "docker" {
```
Run:
```sh
$ docker context ls
NAME              DESCRIPTION                               DOCKER ENDPOINT                             ERROR
default           Current DOCKER_HOST based configuration   unix:///var/run/docker.sock
desktop-linux *   Docker Desktop                            unix:///Users/luk/.docker/run/docker.sock
```
Then I added `host = unix:///Users/luk/.docker/run/docker.sock` to `infrastructure/loadtesting/terraform/init.tf`:
```sh
provider "docker" {
  # Configuration options
  registry_auth {
    address  = "${data.aws_caller_identity.current.account_id}.dkr.ecr.us-east-2.amazonaws.com"
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
  host = "unix:///Users/luk/.docker/run/docker.sock"
}
```