## Terraform

`terraform init && terraform workspace new dev`

`terraform plan`

`terraform apply`

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