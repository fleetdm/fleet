variable "iam_role_arn" {
  type        = string
  description = "IAM Role ARN to use for Kinesis destination logging"
}

variable "kinesis_results_name" {
  type        = string
  description = "name of the kinesis data stream for osquery results logs"
}

variable "kinesis_status_name" {
  type        = string
  description = "name of the kinesis data stream for osquery status logs"
}

variable "kinesis_audit_name" {
  type        = string
  description = "name of the kinesis data stream for fleet audit logs"
}

variable "region" {
  type        = string
  description = "region the target kinesis data stream(s) is in"
}

variable "sts_external_id" {
  type        = string
  description = "Optional unique identifier that can be used by the principal assuming the role to assert its identity."
  default     = ""
}
