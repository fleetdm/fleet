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

#### Database Access
If you need to access the MySQL database backing Fleet Cloud Sandbox, do the following:

1. Obtain database hostname
    ```bash
    aws rds describe-db-clusters --filter Name=db-cluster-id,Values=sandbox-prod --query "DBClusters[0].Endpoint" --output=text
    ```
1. Obtain database master username
    ```bash
    aws rds describe-db-clusters --filter Name=db-cluster-id,Values=sandbox-prod --query "DBClusters[0].MasterUsername" --output=text
    ```
1. Obtain database master password secret name (terraform adds a secret pet name, so we can obtain it from state data)
    ```bash
    terraform show -json | jq -r '.values.root_module.child_modules[].resources | flatten | .[] | select(.address == "module.shared-infrastructure.aws_secretsmanager_secret.database_password_secret").values.name'
    ```
1. Obtain database master password
    ```bash
    aws secretsmanager get-secret-value --secret-id "$(terraform show -json | jq -r '.values.root_module.child_modules[].resources | flatten | .[] | select(.address == "module.shared-infrastructure.aws_secretsmanager_secret.database_password_secret").values.name')" --query "SecretString" --output text
    ```
1. TL;DR -- Put it all together to get into MySQL.  Just copy-paste the part below if you just want the credentials without understanding where they come from.
    ```bash
    DBPASSWORD="$(aws secretsmanager get-secret-value --secret-id "$(terraform show -json | jq -r '.values.root_module.child_modules[].resources | flatten | .[] | select(.address == "module.shared-infrastructure.aws_secretsmanager_secret.database_password_secret").values.name')" --query "SecretString" --output text)"
    aws rds describe-db-clusters --filter Name=db-cluster-id,Values=sandbox-prod --query "DBClusters[0].[Endpoint,MasterUsername]" --output=text | read DBHOST DBUSER
    mysql -h"${DBHOST}" -u"${DBUSER}" -p"${DBPASSWORD}"
    ```

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

#### Cleanup untracked instances fully
This needs to be run in the deprovisioner terraform directory!
```bash
for i in $((aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | .ID.S'; aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | .ID.S'; terraform workspace list | sed 's/ //g' | grep -v '.*default' | sed '/^$/d') | sort | uniq -u); do (terraform workspace select $i && terraform apply -destroy -auto-approve && terraform workspace select default && terraform workspace delete $i); [ $? = 0 ] || break; done
```

#### Useful scripts

1. [tools/upgrade_ecr_ecs.sh](tools/upgrade_ecr_ecs.sh) - Updates the ECR repo with the `FLEET_VERSION` specified and re-runs terraform to ensure the ecs PreProvisioner task uses it in the helm charts.
1. [tools/upgrade_unclaimed.sh](tools/upgrade_unclaimed.sh) - With the changes applied above, this script will replace unclaimed instances with ones upgraded to the new `FLEET_VERSION`.


### Runbooks
#### 5xx errors
If you are seeing 5xx errors, find out what instance its from via the saved query here: https://us-east-2.console.aws.amazon.com/athena/home?region=us-east-2#/query-editor
Make sure you set the workgroup to sandbox-prod-logs otherwise you won't be able to see the saved query.

You can also see errors via the target groups here: https://us-east-2.console.aws.amazon.com/ec2/v2/home?region=us-east-2#TargetGroups:

#### Fleet Logs
Fleet logs can be accessed via kubectl. Setup kubectl by following these instructions: https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html#create-kubeconfig-automatically
Examples:
```bash
# Obtain kubeconfig
aws eks update-kubeconfig --region us-east-2 --name sandbox-prod
# List pods (We currently use the default namespace)
kubectl get pods # Search in there which one it is. There will be 2 instances + a migrations one
# Obtain Logs from all pods for the release. You can also use `--previous` to obtain logs from a previous pod crash if desired.
kubectl logs -l release=<instance id>
```
We do not use eksctl since we use terraform managed resources.

#### Database debugging
Database debugging is accessed through the rds console: https://us-east-2.console.aws.amazon.com/rds/home?region=us-east-2#database:id=sandbox-prod;is-cluster=true
Currently only database metrics are available because performance insights is not available for serverless RDS

If you need to access a specific database for any reason (such as to obtain an email address to reach out in case of an issue), the database name is the same as the instance id.  Using the database access method above, you could use the following example to obtain said email address:

```bash
mysql -h"${DBHOST}" -u"${DBUSER}" -p"${DBPASSWORD}" -D"<instance id>" <<<"SELECT email FROM users;"
``` 
