# AWS S3 File Carving Infrastructure

This Terraform configuration sets up the necessary resources for a secure file carving infrastructure in AWS. File carving is a significant capability for security and forensic analysis, enabling organizations to extract and analyze the content of files from their endpoints.

## Overview of Resources

The resources configured in this Terraform script include:

- **AWS Key Management Service (KMS) Key**: A customer-managed KMS key is created to provide server-side encryption for the S3 bucket where carved files will be stored. The policy attached to this key grants full KMS permissions to the AWS account's root user.

- **Amazon S3 Bucket**: An S3 bucket is provisioned to act as the central repository for storing the results of the file carving process. The bucket is named according to the provided variable `var.bucket_name`.

- **S3 Bucket Server-Side Encryption Configuration**: This resource configures server-side encryption for the S3 bucket, specifying the custom-created KMS key as the master key for encrypting objects stored in the bucket.

- **IAM Policy**: An IAM policy is created to enable specific access to the S3 bucket. This policy is defined via a detailed policy document which grants permissions to perform various actions essential for managing the file carving process. Actions include object retrieval (`GetObject*`), object creation (`PutObject*`), listing the bucket (`ListBucket*`), and managing multipart uploads. It also allows for certain KMS actions necessary for encrypting and decrypting the stored data.

- **IAM Role**: An IAM role (`aws_iam_role`) is provisioned with a trust relationship policy that permits an external entity, specified by `var.fleet_iam_role_arn`, to assume the role. This allows secure access to the S3 bucket and KMS key based on assuming roles across AWS accounts or services.

- **IAM Role Policy Attachment**: This attachment links the previously created IAM policy to the IAM role, ensuring that the permissions are in effect when the role is assumed by the external entity.

## Usage

To use this Terraform configuration, ensure that you have Terraform installed and configured with the necessary AWS credentials. You should define the `bucket_name` and `fleet_iam_role_arn` variables according to your organization's requirements before applying the Terraform plan.

This infrastructure enables secure storage and access for file carving results, facilitating forensic analysis and the capability to respond to security incidents effectively.

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
| [aws_iam_policy.s3_access_policy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.carve_s3_delegation_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.s3_access_attachment](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_kms_key.s3_encryption_key](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_s3_bucket.carve_results_bucket](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket) | resource |
| [aws_s3_bucket_public_access_block.carve_results](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_public_access_block) | resource |
| [aws_s3_bucket_server_side_encryption_configuration.sse](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_server_side_encryption_configuration) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_iam_policy_document.assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.kms_key_policy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.s3_policy](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_bucket_name"></a> [bucket\_name](#input\_bucket\_name) | The name of the osquery carve results bucket | `string` | n/a | yes |
| <a name="input_fleet_iam_role_arn"></a> [fleet\_iam\_role\_arn](#input\_fleet\_iam\_role\_arn) | The IAM role ARN of the Fleet service | `string` | n/a | yes |
| <a name="input_sts_external_id"></a> [sts\_external\_id](#input\_sts\_external\_id) | Optional unique identifier that can be used by the principal assuming the role to assert its identity. | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_iam_role_arn"></a> [iam\_role\_arn](#output\_iam\_role\_arn) | n/a |
| <a name="output_s3_bucket_name"></a> [s3\_bucket\_name](#output\_s3\_bucket\_name) | n/a |
| <a name="output_s3_bucket_region"></a> [s3\_bucket\_region](#output\_s3\_bucket\_region) | n/a |
