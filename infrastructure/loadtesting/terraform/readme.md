## Terraform for Loadtesting Environment

The interface into this code is designed to be minimal.
If you require changes beyond whats described here, contact @zwinnerman-fleetdm.

### Deploying your code to the loadtesting environment
1. Push your branch to https://github.com/fleetdm/fleet and wait for the build to complete (https://github.com/fleetdm/fleet/actions)
1. Initialize your terraform environment with `terraform init`
1. Select a workspace for your test: `terraform workspace new WORKSPACE_NAME; terraform workspace select WORKSPACE_NAME`
1. Apply terraform with your branch name with `terraform apply -var tag=BRANCH_NAME`
1. Perform your tests
1. Clean up the environment with `terraform destroy`

### Running migrations
After applying terraform with the commands above:
`aws ecs run-task --region us-east-2 --cluster fleet-"$(terraform workspace show)"-backend --task-definition fleet-"$(terraform workspace show)"-migrate:"$(terraform output -raw fleet_migration_revision)" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="$(terraform output -raw fleet_migration_subnets)",securityGroups="$(terraform output -raw fleet_migration_security_groups)"}"`

### Running a loadtest
We run simulated hosts in containers of 5,000 at a time. Once the infrastructure is running, you can run the following command:

`terraform apply -var tag=BRANCH_NAME -var loadtest_containers=8`

With the variable `loadtest_containers` you can specify how many containers of 5,000 hosts you want to start. In the example above, it will run 40,000.
