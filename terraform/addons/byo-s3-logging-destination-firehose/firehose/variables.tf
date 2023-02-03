variable "results_destination_s3_bucket" {
  type = string
  description = "s3 bucket name for osquery results"
}

variable "status_destination_s3_bucket" {
  type = string
  description = "s3 bucket name for osquery status"
}

variable "kms_key_arn" {
  type = string
  description = "kms key arn used to encrypt destination buckets"
  default = "arn:aws:kms:us-east-2:123456789123:key/fix-me"
}

variable "firehose_results_name" {
  type = string
  description = "name of the firehose delivery stream for osquery results logs"
}

variable "firehose_status_name" {
  type = string
  description = "name of the firehose delivery stream for osquery status logs"
}

variable "customer_prefix" {
  type = string
  description = "customer prefix to use to namespace all resources"
}

variable "results_object_prefix" {
  type = string
  description = "object prefix for results logs e.g. 'results/'"
  default = "results/"
}

variable "status_object_prefix" {
  type = string
  description = "object prefix for results logs e.g. 'status/'"
  default = "status/"
}