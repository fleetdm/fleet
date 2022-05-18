## Terraform for Loadtesting Environment

The interface into this code is designed to be minimal.
If you require changes beyond whats described here, contact @zwinnerman-fleetdm.

### Deploying your code to the loadtesting environment

1. Push your branch to https://github.com/fleetdm/fleet and wait for the build to complete (https://github.com/fleetdm/fleet/actions).
1. Initialize your terraform environment with `terraform init`.
1. Select a workspace for your test: `terraform workspace new WORKSPACE-NAME; terraform workspace select WORKSPACE-NAME`. Ensure your `WORKSPACE-NAME` contains only alphanumeric characters and hyphens, as it is used to generate names for AWS resources.
1. Apply terraform with your branch name with `terraform apply -var tag=BRANCH_NAME` and type `yes` to approve execution of the plan. This takes a while to complete (~an hour).
1. Perform your tests (see next sections). Your deployment will be available at `https://WORKSPACE-NAME.loadtest.fleetdm.com`.
1. When you're done, clean up the environment with `terraform destroy`.

### Running migrations

After applying terraform with the commands above and before performing your tests, run the following command:
`aws ecs run-task --region us-east-2 --cluster fleet-"$(terraform workspace show)"-backend --task-definition fleet-"$(terraform workspace show)"-migrate:"$(terraform output -raw fleet_migration_revision)" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="$(terraform output -raw fleet_migration_subnets)",securityGroups="$(terraform output -raw fleet_migration_security_groups)"}"`

### Running a loadtest

We run simulated hosts in containers of 5,000 at a time. Once the infrastructure is running, you can run the following command:

`terraform apply -var tag=BRANCH_NAME -var loadtest_containers=8`

With the variable `loadtest_containers` you can specify how many containers of 5,000 hosts you want to start. In the example above, it will run 40,000.

### Monitoring the infrastructure

There are a few main places of interest to monitor the load and resource usage:

* The Application Performance Monitoring (APM) dashboard: access it on your Fleet load-testing URL on port `:5601` and path `/app/apm`, e.g. `https://loadtest.fleetdm.com:5601/app/apm`.
* To monitor mysql database load, go to AWS RDS, select "Performance Insights" and the database instance to monitor (you may want to turn off auto-refresh).
* To monitor Redis load, go to Amazon ElastiCache, select the redis cluster to monitor, and go to "Metrics".
