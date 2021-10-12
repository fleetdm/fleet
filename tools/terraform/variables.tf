variable "prefix" {
  default = "fleet"
}

variable "domain_fleetdm" {
  default = "dogfood.fleetdm.com"
}

variable "domain_fleetctl" {
  default = "dogfood.fleetctl.com"
}

variable "s3_bucket" {
  default = "fleet-osquery-logs-archive"
}

variable "fleet_image" {
  default = "fleetdm/fleet"
}

variable "fleet_min_capacity" {
  default = 1
}

variable "fleet_max_capacity" {
  default = 5
}

variable "osquery_host_count" {
  default = 50
}

variable "vulnerabilities_path" {
  default = "/home/fleet"
}

variable "software_inventory" {
  default = "1"
}

variable "fleet_backend_cpu" {
  default = 256
  type = number
}

variable "fleet_backend_mem" {
  default = 512
  type = number
}

variable "mysql_instance" {
  default = "db.t4g.medium"
}

variable "redis_instance" {
  default = "cache.m5.large"
}

variable "async_host_processing" {
  default = "false"
}

variable "logging_debug" {
  default = "false"
}