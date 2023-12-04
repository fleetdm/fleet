variable "osquery_logging_destination_bucket_name" {
  type        = string
  description = "name of the bucket to store osquery results & status logs"
}

variable "firehose_results_name" {
  type        = string
  description = "firehose delivery stream name for osquery results logs"
  default     = "osquery_results"
}

variable "firehose_status_name" {
  type        = string
  description = "firehose delivery stream name for osquery status logs"
  default     = "osquery_status"
}

variable "firehose_audit_name" {
  type        = string
  description = "firehose delivery stream name for Fleet audit logs"
  default     = "fleet_audit"
}

variable "fleet_iam_role_arn" {
  type        = string
  description = "the arn of the fleet role that firehose will assume to write data to your bucket"
}

variable "results_prefix" {
  default     = "results/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
  description = "s3 object prefix to give to results logs"
}

variable "results_error_prefix" {
  default     = "results/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
  description = "s3 object prefix to give firehose results error logs"
}

variable "status_prefix" {
  default     = "status/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
  description = "s3 object prefix to give status logs"
}

variable "status_error_prefix" {
  default     = "status/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
  description = "s3 object prefix to give firehose status error logs"
}

variable "audit_prefix" {
  default     = "audit/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
  description = "s3 object prefix to give Fleet audit logs"
}

variable "audit_error_prefix" {
  default     = "audit/error/error=!{firehose:error-output-type}/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
  description = "s3 object prefix to give firehose audit error logs"
}

variable "results_buffering_size" {
  type        = number
  default     = 20
  description = "size of the buffer in megabytes before messages are flushed to S3"
}

variable "results_buffering_interval" {
  type        = number
  default     = 120
  description = "size of the time buffer in seconds before messages are flushed to S3"
}

variable "results_compression_format" {
  type    = string
  default = "UNCOMPRESSED"
}

variable "status_buffering_size" {
  type        = number
  default     = 20
  description = "size of the buffer in megabytes before messages are flushed to S3"
}

variable "status_buffering_interval" {
  type        = number
  default     = 120
  description = "size of the time buffer in seconds before messages are flushed to S3"
}

variable "status_compression_format" {
  type    = string
  default = "UNCOMPRESSED"
}

variable "audit_buffering_size" {
  type        = number
  default     = 20
  description = "size of the buffer in megabytes before messages are flushed to S3"
}

variable "audit_buffering_interval" {
  type        = number
  default     = 120
  description = "size of the time buffer in seconds before messages are flushed to S3"
}

variable "audit_compression_format" {
  type    = string
  default = "UNCOMPRESSED"
}