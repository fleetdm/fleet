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
  default     = ""
}

variable "fleet_iam_role_arn" {
  type        = string
  description = "the arn of the fleet role that firehose will assume to write data to your bucket"
}

variable "results_prefix" {
  default     = "results/"
  description = "s3 object prefix to give to results logs"
}

variable "status_prefix" {
  default     = "status/"
  description = "s3 object prefix to give status logs"
}

variable "audit_prefix" {
  default     = "audit/"
  description = "s3 object prefix to give Fleet audit logs"
}