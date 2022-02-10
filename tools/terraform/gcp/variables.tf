variable "region" {
  description = "gcp region"
  default = "us-central1"
}

variable "db_zone" {
  default = "us-central1-c"
}

variable "db_user" {
  default = "fleet"
}

variable "db_tier" {
  default = "db-n1-standard-1"
}

variable "db_version" {
  default = "MYSQL_5_6"
}

variable "serverless_connector_min_instances" {
  default = 2
}
variable "serverless_connector_max_instances" {
  default = 3
}

variable "serverless_connector_instance_type" {
  default = "f1-micro"
}

variable "vpc_subnet" {
  default = "10.10.10.0/28"
}

variable "project_id" {
  description = "gcp project id"
}

variable "prefix" {
  default = "fleet-"
  description = "prefix resources with this string"
}