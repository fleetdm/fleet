## Terraform

`terraform init && terraform workspace new dev`

`terraform plan`

`terraform apply`

### Migrating the DB

After applying terraform run:
```
aws ecs run-task --cluster fleet-backend --task-definition fleet-migrate:<latest_version> --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets=[<private_subnet_id>],securityGroups=[<desired_security_group>]}"
```
