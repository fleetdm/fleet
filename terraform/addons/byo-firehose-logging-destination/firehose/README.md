# Logging Destination: Firehose
This addon provides a Kinesis Firehose logging destination for Fleet with support for cross account S3 delivery.

## Requirements

Apply module `target-account` to provision destination firehose, bucket, kms key, and IAM role/policies.

## Providers

| Name                                              | Version |
|---------------------------------------------------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.52.0  |

## Modules

No modules.

## Resources

No resources.

## Inputs

| Name                          | Description                                               | Type     | Default             | Required |
|-------------------------------|-----------------------------------------------------------|----------|---------------------|:--------:|
| firehose_results_name         | n/a                                                       | `string` | no default provided |   yes    |
| firehose_status_name          | n/a                                                       | `string` | no default provided |   yes    |
| iam_role_arn                  | IAM Role used to write to target firehose delivery stream | `string` | no default provided |   yes    |


## Outputs

| Name                                                                                                            | Description |
|-----------------------------------------------------------------------------------------------------------------|-------------|
| <a name="output_fleet-extra-env-variables"></a> [fleet-extra-env-variables](#output\_fleet-extra-env-variables) | n/a         |
