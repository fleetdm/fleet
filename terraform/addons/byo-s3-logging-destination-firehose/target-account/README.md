# Logging Destination: S3
This module will provision necessary resources to feed osquery results/status logs into S3.

## Requirements

None

## Providers

| Name                                              | Version |
|---------------------------------------------------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.52.0  |

## Modules

No modules.

## Resources

| Name                                                                                                                                                                                             | Type        |
|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------|
| [aws_s3_bucket.osquery-results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket)                                                                           | resource    |
| [aws_s3_bucket.osquery-status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket)                                                                            | resource    |
| [aws_s3_bucket_acl.osquery-results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_acl)                                                                   | resource    |
| [aws_s3_bucket_acl.osquery-status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_acl)                                                                    | resource    |
| [aws_s3_bucket_public_access_block.osquery-results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_public_access_block)                                   | resource    |
| [aws_s3_bucket_public_access_block.osquery-status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_public_access_block)                                    | resource    |
| [aws_s3_bucket_server_side_encryption_configuration.osquery-results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_server_side_encryption_configuration) | resource    |
| [aws_s3_bucket_server_side_encryption_configuration.osquery-status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_server_side_encryption_configuration)  | resource    |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region)                                                                                      | data source |

## Inputs

| Name                   | Description                            | Type     | Default             | Required |
|------------------------|----------------------------------------|----------|---------------------|:--------:|
| osquery_results_bucket | name of the bucket for results logging | `string` | no default provided |   yes    |
| osquery_status_bucket  | name of the bucket for status logging  | `string` | no default provided |   yes    |
| fleet_iam_role_arn     | the role ARN from Fleet Cloud          | `string` | no default provided |   yes    |

## Outputs

| Name                | Description |
|---------------------|-------------|
| kms_key_arn         | n/a         |
| results_bucket_name | n/a         |
| status_bucket_name  | n/a         |
