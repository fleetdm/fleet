## Terraform

### Bootstrapping remote state

First we need to bootstrap our terraform remote state management. This lives outside the main project to avoid "chicken before the egg"
issues. We are going to create the remote state S3 bucket and DynamoDB state locking table and then use hardcoded values
in parent folder `main.tf`.
1. `cd remote-state`
2. `terraform init`
3. `terraform apply`

### Creating the Fleet infrastructure

Create a new `tfvars` file for example:

```terraform
fleet_backend_cpu  = 512
fleet_backend_mem  = 4096 // 4GB needed for vuln processing
redis_instance     = "cache.t3.micro"
fleet_min_capacity = 2
fleet_max_capacity = 5
```

If you have a Fleet license key you can include it in the `tfvars` file which will enable the paid features.

```terraform
fleet_license = "<your license key here"
```

**To deploy the infrastructure**:
1. `terraform init && terraform workspace new prod` (workspace is optional terraform defaults to the `default` workspace)
2. `terraform plan -var-file=<your_tfvars_file>`
3. `terraform apply -var-file=<your_tfvars_file>`

**To deploy cloudwatch alarms** (requires infrastruture to be deployed)
1. `cd monitoring`
2. `terraform init && terraform workspace new prod` (workspace is optional terraform defaults to the `default` workspace)
3. `terraform plan -var-file=<your_tfvars_file>`
4. `terraform apply -var-file=<your_tfvars_file>`

Check out [AWS Chatbot](https://docs.aws.amazon.com/chatbot/latest/adminguide/setting-up.html) for a quick and easy way to hook up Cloudwatch Alarms into a Slack channel. 

### Configuration

Typical settings to override in an existing environment:

`module.vpc.vpc_id` -- the VPC ID output from VPC module. If you are introducing fleet to an existing VPC, you could replace all instances with your VPC ID.

In this reference architecture we are placing ECS, RDS MySQL, and Redis (ElastiCache) in separate subnets, each associated to a route table, allowing communication between.
This is not required, as long as Fleet can resolve the MySQL and Redis hosts, that should be adequate.

#### HTTPS

The ALB is in the public subnet with an ENI to bridge into the private subnet. SSL is terminated at the ALB and `fleet serve` is launched with `FLEET_SERVER_TLS=false` as an
environment variable.

Replace `cert_arn` with the **certificate ARN** that applies to your environment. This is the **certificate ARN** used in the **ALB HTTPS Listener**.

### Migrating the DB

After applying terraform run the following to migrate the database:
```
aws ecs run-task --cluster fleet-backend --task-definition fleet-migrate:<latest_version> --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets=[<private_subnet_id>],securityGroups=[<desired_security_group>]}"
```

### Connecting a Host

Build orbit: 

```
 fleetctl package --type=msi --fleet-url=<alb_dns> --enroll-secret=<secret>
```

Run orbit:

```
 "C:\Program Files\Orbit\bin\orbit\orbit.exe" --root-dir "C:\Program Files\Orbit\." --log-file "C:\Program Files\Orbit\orbit-log.txt" --fleet-url "http://<alb_dns>" --enroll-secret-path "C:\Program Files\Orbit\secret.txt" --update-url "https://tuf.fleetctl.com"  --orbit-channel "stable" --osqueryd-channel "stable"
```