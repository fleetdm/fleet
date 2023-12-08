variable "osquery_logging_destination_bucket_name" {
  type        = string
  description = "name of the bucket to store osquery results & status logs"
}

variable "fleet_iam_role_arn" {
  type        = string
  description = "the arn of the fleet role that firehose will assume to write data to your bucket"
}

variable "log_destinations" {
  description = "A map of configurations for Firehose delivery streams."
  type = map(object({
    name                  = string
    prefix                = string
    error_output_prefix   = string
    buffering_size        = number
    buffering_interval    = number
    compression_format    = string
  }))
  default = {
    results = {
      name                = "osquery_results"
      prefix              = "results/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
      error_output_prefix = "results/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
      buffering_size      = 20
      buffering_interval  = 120
      compression_format  = "UNCOMPRESSED"
    },
    status = {
      name                = "osquery_status"
      prefix              = "status/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
      error_output_prefix = "status/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
      buffering_size      = 20
      buffering_interval  = 120
      compression_format  = "UNCOMPRESSED"
    },
    audit = {
      name                = "fleet_audit"
      prefix              = "audit/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
      error_output_prefix = "audit/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
      buffering_size      = 20
      buffering_interval  = 120
      compression_format  = "UNCOMPRESSED"
    }
  }
}

variable "server_side_encryption_enabled" {
  description = "A boolean flag to enable/disable server-side encryption. Defaults to true (enabled)."
  type        = bool
  default     = true
}

variable "kms_key_arn" {
  description = "An optional KMS key ARN for server-side encryption. If not provided and encryption is enabled, a new key will be created."
  type        = string
  default     = ""
}