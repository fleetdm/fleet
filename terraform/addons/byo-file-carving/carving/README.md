# S3 File Carving backend

This module creates the necessary IAM role for Fleet to attach when it's running in server mode.

It also exports the `fleet_extra_environment_variables` to configure Fleet server to use S3 as the backing carve results store.

Usage typically looks like:

```terraform
fleet_config = {
  extra_environment_variables = merge(
    local.extra_environment_variables,
    module.carving.fleet_extra_environment_variables
  )
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
| [aws_iam_policy.fleet-assume-role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy_document.fleet-assume-role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_iam_role_arn"></a> [iam\_role\_arn](#input\_iam\_role\_arn) | IAM Role ARN to assume into for file carving uploads to S3 | `string` | n/a | yes |
| <a name="input_s3_bucket_name"></a> [s3\_bucket\_name](#input\_s3\_bucket\_name) | The S3 bucket for carve results to be written to | `string` | n/a | yes |
| <a name="input_s3_bucket_region"></a> [s3\_bucket\_region](#input\_s3\_bucket\_region) | The S3 bucket region | `string` | n/a | yes |
| <a name="input_s3_carve_prefix"></a> [s3\_carve\_prefix](#input\_s3\_carve\_prefix) | The S3 object prefix to use when storing carve results | `string` | `""` | no |
| <a name="input_sts_external_id"></a> [sts\_external\_id](#input\_sts\_external\_id) | Optional unique identifier that can be used by the principal assuming the role to assert its identity. | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_fleet_extra_environment_variables"></a> [fleet\_extra\_environment\_variables](#output\_fleet\_extra\_environment\_variables) | n/a |
| <a name="output_fleet_extra_iam_policies"></a> [fleet\_extra\_iam\_policies](#output\_fleet\_extra\_iam\_policies) | n/a |
