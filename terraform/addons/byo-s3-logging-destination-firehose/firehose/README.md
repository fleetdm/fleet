# Logging Destination: Firehose
This addon provides a Kinesis Firehose logging destination for Fleet with support for cross account S3 delivery.

## Requirements

Apply module `target-account` to provision destination bucket, kms key, and IAM policies.

## Providers

| Name                                              | Version |
|---------------------------------------------------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.49.0  |

## Modules

No modules.

## Resources

| Name                                                                                                                                                                 | Type        |
|----------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------|
| [aws_iam_policy.firehose-results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy)                                            | resource    |
| [aws_iam_policy.firehose-status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy)                                             | resource    |
| [aws_iam_role.firehose-results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)                                                | resource    |
| [aws_iam_role.firehose-status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)                                                 | resource    |
| [aws_iam_role_policy_attachment.firehose-results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment)            | resource    |
| [aws_iam_role_policy_attachment.firehose-status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment)             | resource    |
| [aws_kinesis_firehose_delivery_stream.osquery_results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream) | resource    |
| [aws_kinesis_firehose_delivery_stream.osquery_status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream)  | resource    |
| [aws_iam_policy_document.osquery_firehose_assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document)           | data source |
| [aws_iam_policy_document.osquery_results_policy_doc](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document)             | data source |
| [aws_iam_policy_document.osquery_status_policy_doc](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document)              | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region)                                                          | data source |

## Inputs

| Name                          | Description                            | Type     | Default             | Required |
|-------------------------------|----------------------------------------|----------|---------------------|:--------:|
| firehose_results_name         | n/a                                    | `string` | no default provided |   yes    |
| firehose_status_name          | n/a                                    | `string` | no default provided |   yes    |
| customer_prefix               | used for resource tagging              | `string` | no default provided |   yes    |
| kms_key_arn                   | key arn used to encrypt target buckets | `string` | no default provided |   yes    |
| results_destination_s3_bucket | bucket name to send osquery results    | `string` | no default provided |   yes    |
| status_destination_s3_bucket  | bucket name to send osquery status     | `string` | no default provided |   yes    |


## Outputs

| Name                                                                                                            | Description |
|-----------------------------------------------------------------------------------------------------------------|-------------|
| <a name="output_fleet-extra-env-variables"></a> [fleet-extra-env-variables](#output\_fleet-extra-env-variables) | n/a         |
