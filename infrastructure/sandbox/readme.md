## Terraform for the Fleet Demo Environment
This folder holds the infrastructure code for Fleet's demo environment.

This readme itself is intended for infrastructure developers. If you aren't an infrastructure developer, please see https://sandbox.fleetdm.com/openapi.json for documentation.

### Instance state machine
```
provisioned -> unclaimed -> claimed -> [destroyed]
```
provisioned means an instance was "terraform apply'ed" but no installers were generated.
unclaimed means its ready for a customer. claimed means its already in use by a customer. [destroyed] isn't a state you'll see in dynamodb, but it means that everything has been torn down.

### Bugs
1. module.shared-infrastructure.kubernetes_manifest.targetgroupbinding is bugged sometimes, if it gives issues just comment it out
1. on a fresh apply, module.shared-infrastructure.aws_acm_certificate.main will have to be targeted first, then a normal apply can follow
1. If errors happen, see if applying again will fix it
1. There is a secret for apple signing whos values are not provided by this code. If you destroy/apply this secret, then it will have to be filled in manually.

### Environment Access
#### AWS SSO Console
1. You will need to be in the group "AWS Sandbox Prod Admins" in the Fleet Google Workspace
1. From Google Apps, select "AWS SSO"
1. Under "AWS Account" select "Fleet Cloud Sandbox Prod"
1. Choose "Management console" under "SandboxProdAdmins"

#### AWS CLI Access
1. Add the following to your `~/.aws/config`:
```
[profile sandbox_prod]
region = us-east-2
sso_start_url = https://d-9a671703a6.awsapps.com/start
sso_region = us-east-2
sso_account_id = 411315989055
sso_role_name = SandboxProdAdmins
```
1. Login to sso on the cli via `aws sso login --profile=sandbox_prod`
1. To automatically use this profile, `export AWS_PROFILE=sandbox_prod`
1. For more help with AWS SSO Configuration see https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html 

#### VPN Access
You will need to be in the proper group in the Fleet Google Workspace to access this environment.  Access to this environment will "just work" once added.

### Maintenance commands
#### Referesh fleet instances
```bash
for i in $(aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | select(.State.S == "unclaimed") | .ID.S'); do helm uninstall $i; aws dynamodb delete-item --table-name sandbox-prod-lifecycle --key "{\"ID\": {\"S\": \"${i}\"}}"; done
```

#### Cleanup instances that are running, but not tracked
```bash
for i in $((aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | .ID.S'; aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | .ID.S'; helm list | tail -n +2 | cut -f 1) | sort | uniq -u); do helm uninstall $i; done
```

#### Cleanup instances that failed to provision
```bash
for i in $(aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | select(.State.S == "provisioned") | .ID.S'); do helm uninstall $i; aws dynamodb delete-item --table-name sandbox-prod-lifecycle --key "{\"ID\": {\"S\": \"${i}\"}}"; done
```

### Runbooks
#### 5xx errors
If you are seeing 5xx errors, find out what instance its from via the saved query here: https://us-east-2.console.aws.amazon.com/athena/home?region=us-east-2#/query-editor
Make sure you set the workgroup to sandbox-prod-logs otherwise you won't be able to see the saved query.

You can also see errors via the target groups here: https://us-east-2.console.aws.amazon.com/ec2/v2/home?region=us-east-2#TargetGroups:

#### Fleet Logs
Fleet logs can be accessed via kubectl. Setup kubectl by following thexe instructions: https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html#create-kubeconfig-automatically
We do not use eksctl since we use terraform managed resources.

#### Database debugging
Database debugging is accessed through the rds console: https://us-east-2.console.aws.amazon.com/rds/home?region=us-east-2#database:id=sandbox-prod;is-cluster=true
Currently only database metrics are available because performance insights is not available for serverless RDS
