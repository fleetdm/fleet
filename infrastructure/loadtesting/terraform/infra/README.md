# Deploy Loadtesting Infrastructure

# Before we begin

Although deployments through the github action should be prioritized, for manual deployments you will need.

- Terraform v1.10.2
- Docker
- Go

Additionally, refer to the [Reference Architecture sizing recommendations](https://fleetdm.com/docs/deploy/reference-architectures#aws) for loadtest infrastructure sizing.

# Deploy with Github Actions (Coming Soon)

## Deploy/Destroy environment with Github Action

1. [Navigate to the github action](https://github.com/fleetdm/fleet/actions/workflows/loadtest-infra.yml)

2. On the top right corner, select the `Run Workflow` dropdown.

3. Fill out the details for the deployment.

4. After all details have been filled out, you will hit the green `Run Workflow` button, directly under the inputs. For `terraform_action` select `Plan`, `Apply`, or `Destroy`.
    - Plan will show you the results of a dry-run
    - Apply will deploy changes to the environment
    - Destroy will destroy your environment

# Deploy environment manually

1. Clone the repository

2. Initialize terraform

    ```sh
    terraform init
    ```

3. Create a new the terraform workspace or select an existing workspace for your environment. The terraform workspace will be used in different area's of Terraform to drive uniqueness and access to the environment.

    ```sh
    terraform workspace new <workspace_name>
    ```

    or, if your workspace already exists

    ```sh
    terraform workspace list
    terraform workspace select <workspace_name>
    ```

4. Ensure that your new or existing workspace is in use.

    ```sh
    terraform workspace show
    ```

5. Deploy the environment (will also trigger migrations automatically)

    > Note: Terraform will prompt you for confirmation to trigger the deployment. If everything looks ok, submitting `yes` will trigger the deployment.

    ```sh
    terraform apply -var=tag=v4.72.0
    ```

    or, you can add the additional supported terraform variables, to overwrite the default values. You can choose which ones are included/overwritten. If a variable is not defined, the default value configured in [./variables.tf](variables.tf) is used.

    Below is an example with all available variables.

    ```sh
    terraform apply -var=tag=v4.72.0 -var=fleet_task_count=20 -var=fleet_task_memory=4096 -var=fleet_task_cpu=512 -var=database_instance_size=db.t4g.large -var=database_instance_count=3 -var=redis_instance_size=cache.t4g.small -var=redis_instance_count=3
    ```

# Destroy environment manually

1. Clone the repository (if not already cloned)

2. Initialize terraform

    ```sh
    terraform init
    ```

3. Select your workspace

    ```sh
    terraform workspace list
    terraform workspace select <workspace_name>
    ```

3. Destroy the environment

    ```sh
    terraform destroy
    ```

# Delete the workspace

Once all resources have been removed from the terraform workspace, remove the terraform workspace.

```sh
terraform workspace delete <workspace_name>
```

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.68.0 |
| <a name="requirement_docker"></a> [docker](#requirement\_docker) | ~> 2.16.0 |
| <a name="requirement_git"></a> [git](#requirement\_git) | ~> 0.1.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 6.14.1 |
| <a name="provider_docker"></a> [docker](#provider\_docker) | 2.16.0 |
| <a name="provider_git"></a> [git](#provider\_git) | 0.1.0 |
| <a name="provider_random"></a> [random](#provider\_random) | 3.7.2 |
| <a name="provider_terraform"></a> [terraform](#provider\_terraform) | n/a |
| <a name="provider_tls"></a> [tls](#provider\_tls) | 4.1.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_acm"></a> [acm](#module\_acm) | terraform-aws-modules/acm/aws | 4.3.1 |
| <a name="module_loadtest"></a> [loadtest](#module\_loadtest) | github.com/fleetdm/fleet-terraform//byo-vpc | tf-mod-root-v1.18.3 |
| <a name="module_logging_alb"></a> [logging\_alb](#module\_logging\_alb) | github.com/fleetdm/fleet-terraform//addons/logging-alb | tf-mod-addon-logging-alb-v1.6.1 |
| <a name="module_logging_firehose"></a> [logging\_firehose](#module\_logging\_firehose) | github.com/fleetdm/fleet-terraform//addons/logging-destination-firehose | tf-mod-addon-logging-destination-firehose-v1.2.4 |
| <a name="module_mdm"></a> [mdm](#module\_mdm) | github.com/fleetdm/fleet-terraform/addons/mdm?depth=1&ref=tf-mod-addon-mdm-v2.0.0 | n/a |
| <a name="module_migrations"></a> [migrations](#module\_migrations) | github.com/fleetdm/fleet-terraform//addons/migrations | tf-mod-addon-migrations-v2.1.0 |
| <a name="module_osquery-carve"></a> [osquery-carve](#module\_osquery-carve) | github.com/fleetdm/fleet-terraform//addons/osquery-carve | tf-mod-addon-osquery-carve-v1.1.1 |
| <a name="module_ses"></a> [ses](#module\_ses) | github.com/fleetdm/fleet-terraform//addons/ses | tf-mod-addon-ses-v1.4.0 |
| <a name="module_vuln-processing"></a> [vuln-processing](#module\_vuln-processing) | github.com/fleetdm/fleet-terraform//addons/external-vuln-scans | tf-mod-addon-external-vuln-scans-v2.3.0 |

## Resources

| Name | Type |
|------|------|
| [aws_ecr_repository.fleet](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecr_repository) | resource |
| [aws_iam_policy.enroll](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy.license](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role_policy_attachment.enroll](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_kms_alias.alias](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_alias) | resource |
| [aws_kms_key.customer_data_key](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_kms_key.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_lb.internal](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lb) | resource |
| [aws_lb_listener.internal](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lb_listener) | resource |
| [aws_lb_target_group.internal](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lb_target_group) | resource |
| [aws_route53_record.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/route53_record) | resource |
| [aws_secretsmanager_secret_version.scep](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret_version) | resource |
| [aws_security_group.internal](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |
| [docker_registry_image.fleet](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs/resources/registry_image) | resource |
| [random_password.challenge](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password) | resource |
| [random_pet.db_secret_postfix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/pet) | resource |
| [tls_private_key.cloudfront_key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key) | resource |
| [tls_private_key.scep_key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key) | resource |
| [tls_self_signed_cert.scep_cert](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/self_signed_cert) | resource |
| [aws_acm_certificate.certificate](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/acm_certificate) | data source |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_ecr_authorization_token.token](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/ecr_authorization_token) | data source |
| [aws_iam_policy_document.enroll](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.license](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |
| [aws_route53_zone.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/route53_zone) | data source |
| [aws_secretsmanager_secret.license](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/secretsmanager_secret) | data source |
| [aws_secretsmanager_secret_version.enroll_secret](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/secretsmanager_secret_version) | data source |
| [docker_registry_image.dockerhub](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs/data-sources/registry_image) | data source |
| [git_repository.tf](https://registry.terraform.io/providers/paultyng/git/latest/docs/data-sources/repository) | data source |
| [terraform_remote_state.shared](https://registry.terraform.io/providers/hashicorp/terraform/latest/docs/data-sources/remote_state) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_database_instance_count"></a> [database\_instance\_count](#input\_database\_instance\_count) | The number of Aurora database instances | `number` | `2` | no |
| <a name="input_database_instance_size"></a> [database\_instance\_size](#input\_database\_instance\_size) | The instance size for Aurora database instances | `string` | `"db.t4g.medium"` | no |
| <a name="input_fleet_task_count"></a> [fleet\_task\_count](#input\_fleet\_task\_count) | The total number (max) that ECS can scale Fleet containers up to | `number` | `5` | no |
| <a name="input_fleet_task_cpu"></a> [fleet\_task\_cpu](#input\_fleet\_task\_cpu) | The CPU configuration for Fleet containers | `number` | `512` | no |
| <a name="input_fleet_task_memory"></a> [fleet\_task\_memory](#input\_fleet\_task\_memory) | The memory configuration for Fleet containers | `number` | `4096` | no |
| <a name="input_redis_instance_count"></a> [redis\_instance\_count](#input\_redis\_instance\_count) | The number of Elasticache nodes | `number` | `3` | no |
| <a name="input_redis_instance_size"></a> [redis\_instance\_size](#input\_redis\_instance\_size) | The instance size for Elasticache nodes | `string` | `"cache.t4g.micro"` | no |
| <a name="input_tag"></a> [tag](#input\_tag) | The tag to deploy. This would be the same as the branch name | `string` | `"v4.72.0"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_ecs_arn"></a> [ecs\_arn](#output\_ecs\_arn) | n/a |
| <a name="output_ecs_cluster"></a> [ecs\_cluster](#output\_ecs\_cluster) | n/a |
| <a name="output_ecs_execution_arn"></a> [ecs\_execution\_arn](#output\_ecs\_execution\_arn) | n/a |
| <a name="output_enroll_secret_arn"></a> [enroll\_secret\_arn](#output\_enroll\_secret\_arn) | n/a |
| <a name="output_internal_alb_dns_name"></a> [internal\_alb\_dns\_name](#output\_internal\_alb\_dns\_name) | n/a |
| <a name="output_kms_key_id"></a> [kms\_key\_id](#output\_kms\_key\_id) | n/a |
| <a name="output_logging_config"></a> [logging\_config](#output\_logging\_config) | n/a |
| <a name="output_security_groups"></a> [security\_groups](#output\_security\_groups) | n/a |
| <a name="output_server_url"></a> [server\_url](#output\_server\_url) | n/a |
