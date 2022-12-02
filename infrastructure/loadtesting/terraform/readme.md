## Terraform for Loadtesting Environment

The interface into this code is designed to be minimal.
If you require changes beyond whats described here, contact @zwinnerman-fleetdm.

### Deploying your code to the loadtesting environment

1. Push your branch to https://github.com/fleetdm/fleet and wait for the build to complete (https://github.com/fleetdm/fleet/actions).
1. arm64 (M1/M2/etc) Mac Only: run `helpers/setup-darwin_arm64.sh` to build terraform plugins that lack arm64 builds in the registry.  Alternatively, you can use the amd64 terraform binary, which works with Rosetta 2.
1. Initialize your terraform environment with `terraform init`.
1. Select a workspace for your test: `terraform workspace new WORKSPACE-NAME; terraform workspace select WORKSPACE-NAME`. Ensure your `WORKSPACE-NAME` contains only alphanumeric characters and hyphens, as it is used to generate names for AWS resources.
1. Apply terraform with your branch name with `terraform apply -var tag=BRANCH_NAME` and type `yes` to approve execution of the plan. This takes a while to complete (many minutes).
1. Run database migrations (see [Running migrations](#running-migrations)).  You will get 500 errors and your containers will not run if you do not do this.
1. Perform your tests (see [Running a loadtest](#running-a-loadtest)). Your deployment will be available at `https://WORKSPACE-NAME.loadtest.fleetdm.com`.
1. When you're done, clean up the environment with `terraform destroy`.  If A destroy fails, see [ECR Cleanup Troubleshooting](#ecr-cleanup-troubleshooting) for the most common reason.

### Running migrations

After applying terraform with the commands above and before performing your tests, run the following command:
`aws ecs run-task --region us-east-2 --cluster fleet-"$(terraform workspace show)"-backend --task-definition fleet-"$(terraform workspace show)"-migrate:"$(terraform output -raw fleet_migration_revision)" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="$(terraform output -raw fleet_migration_subnets)",securityGroups="$(terraform output -raw fleet_migration_security_groups)"}"`

### Running a loadtest

We run simulated hosts in containers of 5,000 at a time. Once the infrastructure is running, you can run the following command:

`terraform apply -var tag=BRANCH_NAME -var loadtest_containers=8`

With the variable `loadtest_containers` you can specify how many containers of 5,000 hosts you want to start. In the example above, it will run 40,000. If the `fleet` instances need special configuration, you can pass them as environment variables to the `fleet_config` terraform variable, which is a map, using the following syntax (note the use of single quotes around the whole `fleet_config` variable assignment, and the use of double quotes inside its map value):

`terraform apply -var tag=BRANCH_NAME -var loadtest_containers=8 -var='fleet_config={"FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING":"host_last_seen=true","FLEET_OSQUERY_ASYNC_HOST_COLLECT_INTERVAL":"host_last_seen=10s"}'`

### Monitoring the infrastructure

There are a few main places of interest to monitor the load and resource usage:

* The Application Performance Monitoring (APM) dashboard: access it on your Fleet load-testing URL on port `:5601` and path `/app/apm`, e.g. `https://loadtest.fleetdm.com:5601/app/apm`.  Note to do this without the VPN you will need to add your public IP Address to the load balancer for TCP Port 5601.  At the time of this writing, [this](https://us-east-2.console.aws.amazon.com/vpc/home?region=us-east-2#SecurityGroup:groupId=sg-0e67d910a662720f8) will take you directly to the security group for the load balancer if logged into the Load Testing account.
* The APM dashboard can also be accessed via private IP over the VPN.  Use the following one-liner to get the URL: `aws ec2 describe-instances --region=us-east-2 | jq -r '.Reservations[].Instances[] | select(.State.Name == "running") | select(.Tags[] | select(.Key == "ansible_playbook_file") | .Value == "elasticsearch.yml") | "http://" + .PrivateIpAddress + ":5601/app/apm"'`.  This connects directly to the EC2 instance and doesn't use the load balancer.  
* To monitor mysql database load, go to AWS RDS, select "Performance Insights" and the database instance to monitor (you may want to turn off auto-refresh).
* To monitor Redis load, go to Amazon ElastiCache, select the redis cluster to monitor, and go to "Metrics".

### Troubleshooting

#### General Troubleshooting

If terraform fails for some reason, you can make it output extra information to `stderr` by setting the `TF_LOG` environment variable to "DEBUG" or "TRACE", e.g.:

`TF_LOG=DEBUG terraform apply ...`

See https://www.terraform.io/internals/debugging for more details.

#### ECR Cleanup Troubleshooting

In a few instances, it is possible for an ECR repository to still have images left, preventing a full `terraform destroy` of a Loadtesting instance.  Use the following one-liner to clean these up before re-running `terraform destroy`:

`REPOSITORY_NAME=fleet-$(terraform workspace show); aws ecr list-images --repository-name ${REPOSITORY_NAME} --query 'imageIds[*]' --output text | while read digest tag; do aws ecr batch-delete-image --repository-name ${REPOSITORY_NAME} --image-ids imageDigest=${digest}; done`
