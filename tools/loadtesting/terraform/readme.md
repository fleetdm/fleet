## Terraform for Loadtesting Environment

The interface into this code is designed to be minimal.
If you require changes beyond whats described here, contact @zwinnerman-fleetdm.

### Deploying your code to the loadtesting environment
1. Initialize your terraform environment with `terraform init`
2. Apply terraform with your branch name with `terraform apply -var tag=BRANCH_NAME`

### Running migrations
After applying terraform with the commands above:
`aws ecs run-task --region us-east-2 --cluster fleet-backend --task-definition fleet-migrate:"$(terraform output -raw fleet_migration_revision)" --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets="$(terraform output -raw fleet_migration_subnets)",securityGroups="$(terraform output -raw fleet_migration_security_groups)"}"`
