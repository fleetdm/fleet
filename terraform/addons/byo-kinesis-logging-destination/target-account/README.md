# Kinesis Data Stream Logging Destination Setup

## How to use

Below is an example of defining the module to use a single Kinesis Data Stream for all log types (results, status, and audit):
```hcl
module "kinesis" {
  source = "github.com/fleetdm/fleet//terraform/addons/byo-kinesis-logging-destination/target-account"

  fleet_iam_role_arn = "arn:aws:iam::123456789:role/fleet-server-role" # this is the ARN of the IAM (ECS-task) role the fleet servers are running as
  log_destinations = {
    test = {
      name                = "unified-log-stream"
      shard_count         = 0 # shard count only matters if `stream_mode` is `PROVISIONED`
      stream_mode         = "ON_DEMAND" # valid values are `ON_DEMAND` or `PROVISIONED`
      retention_period    = 24 # number of hours you want the data to be retained on the stream
      shard_level_metrics = [] # IncomingBytes IncomingRecords OutgoingBytes OutgoingRecords WriteProvisionedThroughputExceeded ReadProvisionedThroughputExceeded IteratorAgeMilliseconds
    }
  }
}

output "kinesis_logging_destination" {
  value = module.kinesis
}
```

If you desired a Kinesis Data Stream per "topic":
```hcl
module "kinesis" {
   source = "github.com/fleetdm/fleet//terraform/addons/byo-kinesis-logging-destination/target-account"

   fleet_iam_role_arn = "arn:aws:iam::123456789:role/fleet-server-role" # this is the ARN of the IAM (ECS-task) role the fleet servers are running as
   log_destinations = {
      results  = {
         name                = "osquery-results"
         shard_count         = 0 # shard count only matters if `stream_mode` is `PROVISIONED`
         stream_mode         = "ON_DEMAND" # valid values are `ON_DEMAND` or `PROVISIONED`
         retention_period    = 24 # number of hours you want the data to be retained on the stream
         shard_level_metrics = [] # IncomingBytes IncomingRecords OutgoingBytes OutgoingRecords WriteProvisionedThroughputExceeded ReadProvisionedThroughputExceeded IteratorAgeMilliseconds
      }
      status  = {
         name                = "osquery-status"
         shard_count         = 0 # shard count only matters if `stream_mode` is `PROVISIONED`
         stream_mode         = "ON_DEMAND" # valid values are `ON_DEMAND` or `PROVISIONED`
         retention_period    = 24 # number of hours you want the data to be retained on the stream
         shard_level_metrics = [] # IncomingBytes IncomingRecords OutgoingBytes OutgoingRecords WriteProvisionedThroughputExceeded ReadProvisionedThroughputExceeded IteratorAgeMilliseconds
      }
      audit  = {
         name                = "fleet-audit"
         shard_count         = 0 # shard count only matters if `stream_mode` is `PROVISIONED`
         stream_mode         = "ON_DEMAND" # valid values are `ON_DEMAND` or `PROVISIONED`
         retention_period    = 24 # number of hours you want the data to be retained on the stream
         shard_level_metrics = [] # IncomingBytes IncomingRecords OutgoingBytes OutgoingRecords WriteProvisionedThroughputExceeded ReadProvisionedThroughputExceeded IteratorAgeMilliseconds
      }
   }
}

output "kinesis_logging_destination" {
   value = module.kinesis
}
```

1. **Variables:**
    - `fleet_iam_role_arn`: A string variable that holds the ARN of the IAM role which will assume the role defined in this module to gain permissions for writing to the Kinesis Data Streams.
    - `sts_external_id`: An optional string variable that can be used as a unique identifier for the principal assuming the role to assert its identity. Default is an empty string.
    - `log_destinations`: A map variable that contains configurations for multiple Kinesis Data Streams. Each stream configuration includes its name, shard count, stream mode, retention period, and shard-level metrics. Default values are provided for three streams: `osquery_results`, `osquery_status`, and `fleet_audit`.

2. **IAM Role:**
    - `aws_iam_role.fleet_role`: Creates an IAM role with a trust policy that allows the specified IAM role ARN (`fleet_iam_role_arn`) to assume this role. If `sts_external_id` is provided, it adds a condition to the trust policy to require this external ID for the assume role operation.

3. **IAM Policy Documents:**
    - `data.aws_iam_policy_document.assume_role`: Defines the assume role policy for the IAM role. It allows the specified actions (`sts:AssumeRole`) for the specified principals.
    - `data.aws_iam_policy_document.kinesis`: Defines the policy document allowing the IAM role to perform Kinesis and KMS actions. It allows the IAM role to describe, put records into the Kinesis streams (defined in `log_destinations`), and use KMS keys for encryption.

4. **IAM Policy and Attachment:**
    - `aws_iam_policy.fleet_kinesis`: Creates an IAM policy using the defined policy document (`data.aws_iam_policy_document.kinesis`).
    - `aws_iam_role_policy_attachment.fleet_kinesis`: Attaches the created IAM policy to the IAM role (`aws_iam_role.fleet_role`).

5. **KMS Key:**
    - `aws_kms_key.kinesis_key`: Creates a KMS key for encrypting the Kinesis Data Streams.

6. **Kinesis Data Streams:**
    - `aws_kinesis_stream.fleet_log_destination`: Provisions Kinesis Data Streams based on the configurations defined in `log_destinations`. Each stream is created with the specified name, encryption type (using the KMS key), shard-level metrics, shard count (if the stream mode is not "ON\_DEMAND"), and stream mode.

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.3.7 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.29.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.51.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_iam_policy.fleet_kinesis](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.fleet_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.fleet_kinesis](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_kinesis_stream.fleet_log_destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_stream) | resource |
| [aws_kms_key.kinesis_key](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [aws_iam_policy_document.assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.kinesis](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_fleet_iam_role_arn"></a> [fleet\_iam\_role\_arn](#input\_fleet\_iam\_role\_arn) | The ARN of the IAM role that will be assuming into the IAM role defined in this module to gain permissions required to write to the Kinesis Data Stream(s). | `string` | n/a | yes |
| <a name="input_log_destinations"></a> [log\_destinations](#input\_log\_destinations) | A map of configurations for Kinesis data streams. | <pre>map(object({<br>    name                = string<br>    shard_count         = number<br>    stream_mode         = string<br>    retention_period    = number<br>    shard_level_metrics = list(string)<br>  }))</pre> | <pre>{<br>  "audit": {<br>    "name": "fleet_audit",<br>    "retention_period": 24,<br>    "shard_count": 0,<br>    "shard_level_metrics": [],<br>    "stream_mode": "ON_DEMAND"<br>  },<br>  "results": {<br>    "name": "osquery_results",<br>    "retention_period": 24,<br>    "shard_count": 0,<br>    "shard_level_metrics": [],<br>    "stream_mode": "ON_DEMAND"<br>  },<br>  "status": {<br>    "name": "osquery_status",<br>    "retention_period": 24,<br>    "shard_count": 0,<br>    "shard_level_metrics": [],<br>    "stream_mode": "ON_DEMAND"<br>  }<br>}</pre> | no |
| <a name="input_sts_external_id"></a> [sts\_external\_id](#input\_sts\_external\_id) | Optional unique identifier that can be used by the principal assuming the role to assert its identity. | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_kinesis_iam_role"></a> [kinesis\_iam\_role](#output\_kinesis\_iam\_role) | n/a |
| <a name="output_kinesis_streams"></a> [kinesis\_streams](#output\_kinesis\_streams) | A map of Kinesis streams with their names and ARNs. |
