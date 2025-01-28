# Firehose Logging Destination Setup

In this Terraform code, we are defining an IAM Role named `fleet_role` in our AWS Account, that will be assumed by the Fleet application we are hosting. We are only allowing this specific IAM Role (identified by its ARN) to perform certain actions on the Firehose service, such as `DescribeDeliveryStream`, `PutRecord`, and `PutRecordBatch`.

The reason we need a local IAM role in your account is so that we can assume role into it, and you have full control over the permissions it has. The associated IAM policy in the same file specifies the minimum allowed permissions.

The Firehose service is KMS encrypted, so the IAM Role we assume into needs permission to the KMS key that is being used to encrypt the data going into Firehose. Additionally, if the data is being delivered to S3, it will also be encrypted with KMS using the AWS S3 KMS key that is managed by AWS. This is because only customer managed keys can be shared across accounts, and the Firehose delivery stream is actually the one writing to S3.

This code sets up a secure and controlled environment for the Fleet application to perform its necessary actions on the Firehose service within your AWS Account.

If you wanted to make changes to the individual files to fit your environment, feel free. However, it's recommended to use a module like the example below for simplicity.

```
module "firehose_logging" {
  source = "github.com/fleetdm/fleet//terraform/addons/byo-firehose-logging-destination/target-account"
  
  # Variables
  osquery_logging_destination_bucket_name = {your-desired-bucket-prefix}
  fleet_iam_role_arn                      = {supplied by Fleet}
  sts_external_id                         = {if using}
}
```
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.7 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.29.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | >= 5.29.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_iam_policy.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy.fleet_firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role.fleet_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.fleet_firehose](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_kinesis_firehose_delivery_stream.fleet_log_destinations](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream) | resource |
| [aws_kms_key.firehose_key](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_s3_bucket.destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket) | resource |
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
| <a name="input_fleet_iam_role_arn"></a> [fleet\_iam\_role\_arn](#input\_fleet\_iam\_role\_arn) | the arn of the fleet role that firehose will assume to write data to your bucket | `string` | n/a | yes |
| <a name="input_kms_key_arn"></a> [kms\_key\_arn](#input\_kms\_key\_arn) | An optional KMS key ARN for server-side encryption. If not provided and encryption is enabled, a new key will be created. | `string` | `""` | no |
| <a name="input_log_destinations"></a> [log\_destinations](#input\_log\_destinations) | A map of configurations for Firehose delivery streams. | <pre>map(object({<br>    name                = string<br>    prefix              = string<br>    error_output_prefix = string<br>    buffering_size      = number<br>    buffering_interval  = number<br>    compression_format  = string<br>  }))</pre> | <pre>{<br>  "audit": {<br>    "buffering_interval": 120,<br>    "buffering_size": 20,<br>    "compression_format": "UNCOMPRESSED",<br>    "error_output_prefix": "audit/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/",<br>    "name": "fleet_audit",<br>    "prefix": "audit/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"<br>  },<br>  "results": {<br>    "buffering_interval": 120,<br>    "buffering_size": 20,<br>    "compression_format": "UNCOMPRESSED",<br>    "error_output_prefix": "results/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/",<br>    "name": "osquery_results",<br>    "prefix": "results/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"<br>  },<br>  "status": {<br>    "buffering_interval": 120,<br>    "buffering_size": 20,<br>    "compression_format": "UNCOMPRESSED",<br>    "error_output_prefix": "status/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/",<br>    "name": "osquery_status",<br>    "prefix": "status/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"<br>  }<br>}</pre> | no |
| <a name="input_osquery_logging_destination_bucket_name"></a> [osquery\_logging\_destination\_bucket\_name](#input\_osquery\_logging\_destination\_bucket\_name) | name of the bucket to store osquery results & status logs | `string` | n/a | yes |
| <a name="input_server_side_encryption_enabled"></a> [server\_side\_encryption\_enabled](#input\_server\_side\_encryption\_enabled) | A boolean flag to enable/disable server-side encryption. Defaults to true (enabled). | `bool` | `true` | no |
| <a name="input_sts_external_id"></a> [sts\_external\_id](#input\_sts\_external\_id) | Optional unique identifier that can be used by the principal assuming the role to assert its identity. | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_firehose_iam_role"></a> [firehose\_iam\_role](#output\_firehose\_iam\_role) | n/a |
| <a name="output_log_destinations"></a> [log\_destinations](#output\_log\_destinations) | Map of Firehose delivery streams' names. |
| <a name="output_s3_destination"></a> [s3\_destination](#output\_s3\_destination) | n/a |
