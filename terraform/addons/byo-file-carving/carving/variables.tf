variable "iam_role_arn" {
  type        = string
  description = "IAM Role ARN to assume into for file carving uploads to S3"
}

variable "sts_external_id" {
  type = string
  description = "Optional unique identifier that can be used by the principal assuming the role to assert its identity."
  default = ""
}

variable "s3_bucket_name" {
  type =  string
  description = "The S3 bucket for carve results to be written to"
}

variable "s3_bucket_region" {
  type = string
  description = "The S3 bucket region"
}

variable "s3_carve_prefix" {
  type = string
  description = "The S3 object prefix to use when storing carve results"
  default = ""
}