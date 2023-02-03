variable "osquery_results_bucket" {
  type        = string
  description = "name of the bucket to store osquery results logs"
}

variable "osquery_status_bucket" {
  type        = string
  description = "name of the bucket to store osquery status logs"
}

variable "fleet_iam_role_arn" {
  type        = string
  description = "the arn of the fleet role that firehose will assume to write data to your bucket"
}