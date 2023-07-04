variable "iam_role_arn" {
  type        = string
  description = "IAM Role ARN to use for Firehose destination logging"
}

variable "firehose_results_name" {
  type        = string
  description = "name of the firehose delivery stream for osquery results logs"
}

variable "firehose_status_name" {
  type        = string
  description = "name of the firehose delivery stream for osquery status logs"
}

variable "firehose_audit_name" {
  type        = string
  description = "name of the firehose delivery stream for fleet audit logs"
}

variable "region" {
  type        = string
  description = "region the target firehose delivery stream is in"
}
