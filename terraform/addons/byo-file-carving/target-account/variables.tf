variable "bucket_name" {
  type = string
  description = "The name of the osquery carve results bucket"
}

variable "fleet_iam_role_arn" {
  type = string
  description = "The IAM role ARN of the Fleet service"
}