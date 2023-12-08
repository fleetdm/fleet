# ALB Logging Addon
This addon creates alb logging bucket(s) in s3 and optionally an athena database for those logs.

# Example Configuration

This assumes your fleet module is `main` and is configured with it's default documentation.

See https://github.com/fleetdm/fleet/blob/main/terraform/example/main.tf for details.

```
module "main" {
  source          = "github.com/fleetdm/fleet//terraform?ref=main"
  certificate_arn = module.acm.acm_certificate_arn
  vpc = {
    name = random_pet.main.id
  }
  fleet_config = {
    extra_environment_variables = module.firehose-logging.fleet_extra_environment_variables
    extra_iam_policies          = module.firehose-logging.fleet_extra_iam_policies
  }
  alb_config = {
    access_logs = {
      bucket  = module.logging_alb.log_s3_bucket_id
      prefix  = "fleet"
      enabled = true
    }
  }
}

module "logging_alb" {
  source        = "github.com/fleetdm/fleet//terraform/addons/logging-alb?ref=main"
  prefix        = "fleet"
  enable_athena = true
}
```

# Additional Information

Once this terraform is applied, the Athena table will need to be created.  See https://docs.aws.amazon.com/athena/latest/ug/application-load-balancer-logs.html for help with creating the table.

For this implementation, the S3 pattern for the `CREATE TABLE` query should look like this:

```
s3://your-alb-logs-bucket/<PREFIX>/AWSLogs/<ACCOUNT-ID>/elasticloadbalancing/<REGION>/
```

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.25.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_athena-s3-bucket"></a> [athena-s3-bucket](#module\_athena-s3-bucket) | terraform-aws-modules/s3-bucket/aws | 3.15.1 |
| <a name="module_s3_bucket_for_logs"></a> [s3\_bucket\_for\_logs](#module\_s3\_bucket\_for\_logs) | terraform-aws-modules/s3-bucket/aws | 3.15.1 |

## Resources

| Name | Type |
|------|------|
| [aws_athena_database.logs](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/athena_database) | resource |
| [aws_athena_workgroup.logs](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/athena_workgroup) | resource |
| [aws_kms_alias.logs_alias](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_alias) | resource |
| [aws_kms_key.logs](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_iam_policy_document.kms](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.s3_athena_bucket](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.s3_log_bucket](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_enable_athena"></a> [enable\_athena](#input\_enable\_athena) | n/a | `bool` | `true` | no |
| <a name="input_extra_kms_policies"></a> [extra\_kms\_policies](#input\_extra\_kms\_policies) | n/a | `list(any)` | `[]` | no |
| <a name="input_extra_s3_athena_policies"></a> [extra\_s3\_athena\_policies](#input\_extra\_s3\_athena\_policies) | n/a | `list(any)` | `[]` | no |
| <a name="input_extra_s3_log_policies"></a> [extra\_s3\_log\_policies](#input\_extra\_s3\_log\_policies) | n/a | `list(any)` | `[]` | no |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | n/a | `string` | `"fleet"` | no |
| <a name="input_s3_expiration_days"></a> [s3\_expiration\_days](#input\_s3\_expiration\_days) | n/a | `number` | `90` | no |
| <a name="input_s3_newer_noncurrent_versions"></a> [s3\_newer\_noncurrent\_versions](#input\_s3\_newer\_noncurrent\_versions) | n/a | `number` | `5` | no |
| <a name="input_s3_noncurrent_version_expiration_days"></a> [s3\_noncurrent\_version\_expiration\_days](#input\_s3\_noncurrent\_version\_expiration\_days) | n/a | `number` | `30` | no |
| <a name="input_s3_transition_days"></a> [s3\_transition\_days](#input\_s3\_transition\_days) | n/a | `number` | `30` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_log_s3_bucket_id"></a> [log\_s3\_bucket\_id](#output\_log\_s3\_bucket\_id) | n/a |
