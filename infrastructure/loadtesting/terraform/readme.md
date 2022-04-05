## Terraform for Loadtesting Environment

The interface into this code is designed to be minimal.
If you require changes beyond whats described here, contact @zwinnerman-fleetdm.

### Deploying your code to the loadtesting environment
1. Push your branch to https://github.com/fleetdm/fleet and wait for the build to complete (https://github.com/fleetdm/fleet/actions)
1. Initialize your terraform environment with `terraform init`
1. Apply terraform with your branch name with `terraform apply -var tag=BRANCH_NAME`

### Running migrations
After applying terraform with the commands above:
`aws ecs run-task --region us-east-2 --cluster fleet-backend --task-definition fleet-migrate:"$(terraform output -raw fleet_migration_revision)" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="$(terraform output -raw fleet_migration_subnets)",securityGroups="$(terraform output -raw fleet_migration_security_groups)"}"`

### Running a loadtest
We run simulated hosts in containers of 5,000 at a time. Once the infrastructure is running, you can run the following command:

`terraform apply -var tag=BRANCH_NAME -var loadtest_containers=8`

With the variable `loadtest_containers` you can specify how many containers of 5,000 hosts you want to start. In the example above, it will run 40,000.
