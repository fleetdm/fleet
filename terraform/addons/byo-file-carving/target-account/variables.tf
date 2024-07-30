variable "bucket_name" {
  type = string
  description = "The name of the osquery carve results bucket"
}

variable "fleet_iam_role_arn" {
  type = string
  description = "The IAM role ARN of the Fleet service"
}

variable "sts_external_id" {
  type        = string
  description = "Optional unique identifier that can be used by the principal assuming the role to assert its identity."
  default     = ""
}