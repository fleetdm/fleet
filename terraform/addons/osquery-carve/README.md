# Osquery Carve Bucket Addon
This addon provides a S3 bucket for Osquery Carve results.

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.39.1 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_iam_policy.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_s3_bucket.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket) | resource |
| [aws_s3_bucket_lifecycle_configuration.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_lifecycle_configuration) | resource |
| [aws_s3_bucket_public_access_block.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_public_access_block) | resource |
| [aws_s3_bucket_server_side_encryption_configuration.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_server_side_encryption_configuration) | resource |
| [aws_iam_policy_document.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_osquery_carve_s3_bucket"></a> [osquery\_carve\_s3\_bucket](#input\_osquery\_carve\_s3\_bucket) | n/a | <pre>object({<br>    name         = optional(string, "fleet-osquery-results-archive")<br>    expires_days = optional(number, 1)<br>  })</pre> | <pre>{<br>  "expires_days": 1,<br>  "name": "fleet-osquery-results-archive"<br>}</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_fleet_extra_environment_variables"></a> [fleet\_extra\_environment\_variables](#output\_fleet\_extra\_environment\_variables) | n/a |
| <a name="output_fleet_extra_iam_policies"></a> [fleet\_extra\_iam\_policies](#output\_fleet\_extra\_iam\_policies) | n/a |
