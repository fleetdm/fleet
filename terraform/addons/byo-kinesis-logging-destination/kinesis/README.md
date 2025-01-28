# Kinesis Data Stream Logging Destination Setup

## Usage

After `./target-account` module is applied you might use this module in the following manner:

```hcl
module "kinesis" {
   source               = "../../../../fleet/terraform/addons/byo-kinesis-logging-destination/kinesis"
   kinesis_results_name = "testing-log-stream"
   kinesis_status_name  = "testing-log-stream"
   kinesis_audit_name   = "testing-log-stream"
   iam_role_arn         = "arn:aws:iam::123456789:role/terraform-20240524165353382600000001"
   region               = "us-east-2"
}
```

And then supply the outputs to the `fleet_config` definition:
```hcl
fleet_config = {
    image = local.fleet_image
    extra_iam_policies = concat(module.kinesis.fleet_extra_iam_policies)
    extra_environment_variables = merge(
      local.extra_environment_variables,
      module.kinesis.fleet_extra_environment_variables,
    )
  }
```

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
| [aws_iam_policy.fleet-assume-role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy_document.fleet-assume-role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_iam_role_arn"></a> [iam\_role\_arn](#input\_iam\_role\_arn) | IAM Role ARN to use for Kinesis destination logging | `string` | n/a | yes |
| <a name="input_kinesis_audit_name"></a> [kinesis\_audit\_name](#input\_kinesis\_audit\_name) | name of the kinesis data stream for fleet audit logs | `string` | n/a | yes |
| <a name="input_kinesis_results_name"></a> [kinesis\_results\_name](#input\_kinesis\_results\_name) | name of the kinesis data stream for osquery results logs | `string` | n/a | yes |
| <a name="input_kinesis_status_name"></a> [kinesis\_status\_name](#input\_kinesis\_status\_name) | name of the kinesis data stream for osquery status logs | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | region the target kinesis data stream(s) is in | `string` | n/a | yes |
| <a name="input_sts_external_id"></a> [sts\_external\_id](#input\_sts\_external\_id) | Optional unique identifier that can be used by the principal assuming the role to assert its identity. | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_fleet_extra_environment_variables"></a> [fleet\_extra\_environment\_variables](#output\_fleet\_extra\_environment\_variables) | n/a |
| <a name="output_fleet_extra_iam_policies"></a> [fleet\_extra\_iam\_policies](#output\_fleet\_extra\_iam\_policies) | n/a |
