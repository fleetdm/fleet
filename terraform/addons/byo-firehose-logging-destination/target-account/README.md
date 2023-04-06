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

| Name                                                                                                                                                                                                 | Type        |
|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------|
| [aws_s3_bucket.osquery-destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket)                                                                           | resource    |
| [aws_s3_bucket_acl.osquery-destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_acl)                                                                   | resource    |
| [aws_s3_bucket_public_access_block.osquery-destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_public_access_block)                                   | resource    |
| [aws_s3_bucket_server_side_encryption_configuration.osquery-destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_server_side_encryption_configuration) | resource    |
| [aws_iam_policy.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy)                                                                                    | resource    |
| [aws_iam_role.fleet_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)                                                                                      | resource    |
| [aws_iam_role.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)                                                                                        | resource    |
| [aws_iam_role_policy_attachment.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment)                                                    | resource    |
| [aws_kinesis_firehose_delivery_stream.osquery_results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream)                                 | resource    |
| [aws_kinesis_firehose_delivery_stream.osquery_status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream)                                  | resource    |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region)                                                                                          | data source |
| [aws_iam_policy_document.osquery_firehose_assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document)                                           | data source |
| [aws_iam_policy_document.osquery_results_policy_doc](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document)                                             | data source |
| [aws_iam_policy_document.osquery_status_policy_doc](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document)                                              | data source |

## Inputs

| Name                                    | Description                                              | Type     | Default             | Required |
|-----------------------------------------|----------------------------------------------------------|----------|---------------------|:--------:|
| osquery_logging_destination_bucket_name | name of the bucket for osquery logging                   | `string` | no default provided |   yes    |
| firehose_results_name                   | name of the firehose delivery stream for results logging | `string` | `osquery_results`   |    no    |
| firehose_status_name                    | name of the firehose delivery stream for status logging  | `string` | `osquery_status`    |    no    |
| results_prefix                          | s3 object prefix to give to results logs                 | `string` | `results/`          |    no    |
| status_prefix                           | s3 object prefix to give status logs                     | `string` | `status/`           |    no    |
| fleet_iam_role_arn                      | the role ARN from Fleet Cloud                            | `string` | no default provided |   yes    |



## Outputs

| Name              | Description                                                                     |
|-------------------|---------------------------------------------------------------------------------|
| firehose_iam_role | IAM Role ARN fleet cloud will assume to write data to firehose delivery streams |
| firehose_results  | name of the firehose delivery stream for results logs                           |
| firehose_status   | name of the firehose delivery stream for status logs                            |
