## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.17.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_saml_auth_proxy_alb"></a> [saml\_auth\_proxy\_alb](#module\_saml\_auth\_proxy\_alb) | terraform-aws-modules/alb/aws | 8.2.1 |

## Resources

| Name | Type |
|------|------|
| [aws_cloudwatch_log_group.saml_auth_proxy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_log_group) | resource |
| [aws_ecs_service.saml_auth_proxy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_service) | resource |
| [aws_ecs_task_definition.saml_auth_proxy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_task_definition) | resource |
| [aws_iam_policy.saml_auth_proxy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_secretsmanager_secret.saml_auth_proxy_cert](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_security_group.saml_auth_proxy_alb](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |
| [aws_security_group.saml_auth_proxy_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |
| [aws_iam_policy_document.saml_auth_proxy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alb_access_logs"></a> [alb\_access\_logs](#input\_alb\_access\_logs) | n/a | `map(string)` | `{}` | no |
| <a name="input_alb_target_group_arn"></a> [alb\_target\_group\_arn](#input\_alb\_target\_group\_arn) | n/a | `string` | n/a | yes |
| <a name="input_base_url"></a> [base\_url](#input\_base\_url) | n/a | `string` | n/a | yes |
| <a name="input_cookie_max_age"></a> [cookie\_max\_age](#input\_cookie\_max\_age) | n/a | `string` | `"1h"` | no |
| <a name="input_customer_prefix"></a> [customer\_prefix](#input\_customer\_prefix) | customer prefix to use to namespace all resources | `string` | `"fleet"` | no |
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster) | n/a | `string` | n/a | yes |
| <a name="input_ecs_execution_iam_role_arn"></a> [ecs\_execution\_iam\_role\_arn](#input\_ecs\_execution\_iam\_role\_arn) | n/a | `string` | n/a | yes |
| <a name="input_ecs_iam_role_arn"></a> [ecs\_iam\_role\_arn](#input\_ecs\_iam\_role\_arn) | n/a | `string` | n/a | yes |
| <a name="input_idp_metadata_url"></a> [idp\_metadata\_url](#input\_idp\_metadata\_url) | n/a | `string` | n/a | yes |
| <a name="input_logging_options"></a> [logging\_options](#input\_logging\_options) | n/a | <pre>object({<br>    awslogs-group         = string<br>    awslogs-region        = string<br>    awslogs-stream-prefix = string<br>  })</pre> | n/a | yes |
| <a name="input_proxy_containers"></a> [proxy\_containers](#input\_proxy\_containers) | n/a | `number` | `1` | no |
| <a name="input_saml_auth_proxy_image"></a> [saml\_auth\_proxy\_image](#input\_saml\_auth\_proxy\_image) | n/a | `string` | `"itzg/saml-auth-proxy:1.12.0@sha256:ddff17caa00c1aad64d6c7b2e1d5eb93d97321c34d8ad12a25cfd8ce203db723"` | no |
| <a name="input_security_groups"></a> [security\_groups](#input\_security\_groups) | n/a | `list(string)` | n/a | yes |
| <a name="input_subnets"></a> [subnets](#input\_subnets) | n/a | `list(string)` | n/a | yes |
| <a name="input_vpc_id"></a> [vpc\_id](#input\_vpc\_id) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_fleet_extra_execution_policies"></a> [fleet\_extra\_execution\_policies](#output\_fleet\_extra\_execution\_policies) | n/a |
| <a name="output_lb"></a> [lb](#output\_lb) | n/a |
| <a name="output_lb_security_group"></a> [lb\_security\_group](#output\_lb\_security\_group) | n/a |
| <a name="output_lb_target_group_arn"></a> [lb\_target\_group\_arn](#output\_lb\_target\_group\_arn) | Keep for legacy support for now |
| <a name="output_name"></a> [name](#output\_name) | n/a |
| <a name="output_secretsmanager_secret_id"></a> [secretsmanager\_secret\_id](#output\_secretsmanager\_secret\_id) | n/a |
