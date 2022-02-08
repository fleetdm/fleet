variable "region" {
  description = "gcp region"
  default = "us-central1"
}

variable "project_id" {
  description = "gcp project id"
}

variable "prefix" {
  default = "fleet-"
  description = "prefix resources with this string"
}