## Introduction

In this Terraform code, we are defining an IAM Role named `fleet_role` in our AWS Account, that will be assumed by the Fleet application we are hosting. We are only allowing this specific IAM Role (identified by its ARN) to perform certain actions on the Firehose service, such as `DescribeDeliveryStream`, `PutRecord`, and `PutRecordBatch`.

The reason we need a local IAM role in your account is so that we can assume role into it, and you have full control over the permissions it has. The associated IAM policy in the same file specifies the minimum allowed permissions.

The Firehose service is KMS encrypted, so the IAM Role we assume into needs permission to the KMS key that is being used to encrypt the data going into Firehose. Additionally, if the data is being delivered to S3, it will also be encrypted with KMS using the AWS S3 KMS key that is managed by AWS. This is because only customer managed keys can be shared across accounts, and the Firehose delivery stream is actually the one writing to S3.

Overall, this code sets up a secure and controlled environment for the Fleet application to perform its necessary actions on the Firehose service within your AWS Account.
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
| [aws_iam_policy.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy.fleet-firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy_attachment.fleet-firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy_attachment) | resource |
| [aws_iam_role.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role.fleet_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_kinesis_firehose_delivery_stream.osquery_results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream) | resource |
| [aws_kinesis_firehose_delivery_stream.osquery_status](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream) | resource |
| [aws_kms_key.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_s3_bucket.destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket) | resource |
| [aws_s3_bucket_acl.destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_acl) | resource |
| [aws_s3_bucket_public_access_block.destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_public_access_block) | resource |
| [aws_s3_bucket_server_side_encryption_configuration.destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_server_side_encryption_configuration) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_iam_policy_document.assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.firehose_policy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.osquery_firehose_assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_kms_alias.s3](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/kms_alias) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_firehose_results_name"></a> [firehose\_results\_name](#input\_firehose\_results\_name) | firehose delivery stream name for osquery results logs | `string` | `"osquery_results"` | no |
| <a name="input_firehose_status_name"></a> [firehose\_status\_name](#input\_firehose\_status\_name) | firehose delivery stream name for osquery status logs | `string` | `"osquery_status"` | no |
| <a name="input_fleet_iam_role_arn"></a> [fleet\_iam\_role\_arn](#input\_fleet\_iam\_role\_arn) | the arn of the fleet role that firehose will assume to write data to your bucket | `string` | n/a | yes |
| <a name="input_osquery_logging_destination_bucket_name"></a> [osquery\_logging\_destination\_bucket\_name](#input\_osquery\_logging\_destination\_bucket\_name) | name of the bucket to store osquery results & status logs | `string` | n/a | yes |
| <a name="input_results_prefix"></a> [results\_prefix](#input\_results\_prefix) | s3 object prefix to give to results logs | `string` | `"results/"` | no |
| <a name="input_status_prefix"></a> [status\_prefix](#input\_status\_prefix) | s3 object prefix to give status logs | `string` | `"status/"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_firehose_iam_role"></a> [firehose\_iam\_role](#output\_firehose\_iam\_role) | n/a |
| <a name="output_firehose_results"></a> [firehose\_results](#output\_firehose\_results) | n/a |
| <a name="output_firehose_status"></a> [firehose\_status](#output\_firehose\_status) | n/a |
| <a name="output_s3_destination"></a> [s3\_destination](#output\_s3\_destination) | n/a |
<!-- END_TF_DOCS -->