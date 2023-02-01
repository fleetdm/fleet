# osquery-perf addon
This addon adds osquery-perf hosts to the Fleet installation.
These are generally used for loadtesting or other testing purposes.  See https://github.com/fleetdm/fleet/tree/main/cmd/osquery-perf to learn more about osquery-perf itself.

This addon creates an AWS Secrets Manager secret that will be used to store the enroll secret that the osquery-perf hosts use to enroll into Fleet.  This secret will need to have its `SecretString` populated with the enroll secret manually once everything is setup in order for the osquery-perf hosts to connect.

Below is an example implementation of the module:

```
module "osquery_perf" {
  source                     = "github.com/fleetdm/fleet//terraform/addons/osquery-perf?ref=main"
  customer_prefix            = "fleet"
  ecs_cluster                = module.main.byo-vpc.byo-db.byo-ecs.service.cluster
  subnets                    = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups            = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
  ecs_iam_role_arn           = module.main.byo-vpc.byo-db.byo-ecs.iam_role_arn
  ecs_execution_iam_role_arn = module.main.byo-vpc.byo-db.byo-ecs.execution_iam_role_arn
  server_url                 = "https://${aws_route53_record.main.fqdn}"
  osquery_perf_image         = local.osquery_perf_image
  extra_flags                = ["--os_templates", "mac10.14.6,ubuntu_22.04,windows_11"]
  logging_options            = module.main.byo-vpc.byo-db.byo-ecs.logging_config
}
```

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_ecs_service.osquery_perf](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_service) | resource |
| [aws_ecs_task_definition.osquery_perf](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_task_definition) | resource |
| [aws_kms_alias.enroll_secret](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_alias) | resource |
| [aws_kms_key.enroll_secret](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_secretsmanager_secret.enroll_secret](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_secretsmanager_secret_version.enroll_secret](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/secretsmanager_secret_version) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_customer_prefix"></a> [customer\_prefix](#input\_customer\_prefix) | customer prefix to use to namespace all resources | `string` | `"fleet"` | no |
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster) | n/a | `string` | n/a | yes |
| <a name="input_ecs_execution_iam_role_arn"></a> [ecs\_execution\_iam\_role\_arn](#input\_ecs\_execution\_iam\_role\_arn) | n/a | `string` | n/a | yes |
| <a name="input_ecs_iam_role_arn"></a> [ecs\_iam\_role\_arn](#input\_ecs\_iam\_role\_arn) | n/a | `string` | n/a | yes |
| <a name="input_extra_flags"></a> [extra\_flags](#input\_extra\_flags) | n/a | `list(string)` | `[]` | no |
| <a name="input_loadtest_containers"></a> [loadtest\_containers](#input\_loadtest\_containers) | n/a | `number` | `1` | no |
| <a name="input_logging_options"></a> [logging\_options](#input\_logging\_options) | n/a | <pre>object({<br>    awslogs-group         = string<br>    awslogs-region        = string<br>    awslogs-stream-prefix = string<br>  })</pre> | n/a | yes |
| <a name="input_osquery_perf_image"></a> [osquery\_perf\_image](#input\_osquery\_perf\_image) | n/a | `string` | n/a | yes |
| <a name="input_security_groups"></a> [security\_groups](#input\_security\_groups) | n/a | `list(string)` | n/a | yes |
| <a name="input_server_url"></a> [server\_url](#input\_server\_url) | n/a | `string` | n/a | yes |
| <a name="input_subnets"></a> [subnets](#input\_subnets) | n/a | `list(string)` | n/a | yes |

## Outputs

No outputs.
