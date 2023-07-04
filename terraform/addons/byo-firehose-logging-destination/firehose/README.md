<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.7 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 4.52.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | >= 4.52.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_iam_policy.fleet-assume-role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy_document.fleet-assume-role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_firehose_results_name"></a> [firehose\_results\_name](#input\_firehose\_results\_name) | name of the firehose delivery stream for osquery results logs | `string` | n/a | yes |
| <a name="input_firehose_status_name"></a> [firehose\_status\_name](#input\_firehose\_status\_name) | name of the firehose delivery stream for osquery status logs | `string` | n/a | yes |
| <a name="input_iam_role_arn"></a> [iam\_role\_arn](#input\_iam\_role\_arn) | IAM Role ARN to use for Firehose destination logging | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_fleet_extra_environment_variables"></a> [fleet\_extra\_environment\_variables](#output\_fleet\_extra\_environment\_variables) | n/a |
| <a name="output_fleet_extra_iam_policies"></a> [fleet\_extra\_iam\_policies](#output\_fleet\_extra\_iam\_policies) | n/a |
<!-- END_TF_DOCS -->