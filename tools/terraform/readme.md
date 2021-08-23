## Terraform

`terraform init && terraform workspace new dev`

`terraform plan`

`terraform apply`

### Migrating the DB

After applying terraform run:
```
aws ecs run-task --cluster fleet-backend --task-definition fleet-migrate:<latest_version> --launch-type FARGATE --network-configuration "awsvpcConfiguration={subnets=[<private_subnet_id>],securityGroups=[<desired_security_group>]}"
```

### Connecting a Host

Build orbit: 

```
 go run ./cmd/package --type=msi --fleet-url=<alb_dns> --insecure --enroll-secret=<secret>
```

Run orbit:

```
 "C:\Program Files\Orbit\bin\orbit\orbit.exe" --root-dir "C:\Program Files\Orbit\." --log-file "C:\Program Files\Orbit\orbit-log.txt" --fleet-url "http://<alb_dns>" --enroll-secret-path "C:\Program Files\Orbit\secret.txt" --insecure --update-url "https://tuf.fleetctl.com"  --orbit-channel "stable" --osqueryd-channel "stable"
```