locals {
  name = "fleetdm"
}

variable "prefix" {
  default = "fleet"
}

variable "domain_fleetdm" {
  default = "dogfood.fleetdm.com"
}

variable "domain_fleetctl" {
  default = "dogfood.fleetctl.com"
}

variable "database_user" {
  description = "database user fleet will authenticate and query with"
  default     = "fleet"
}

variable "database_name" {
  description = "the name of the database fleet will create/use"
  default = "fleet"
}

variable "image" {
  description = "the name of the container image to run"
  default     = "fleetdm/fleet"
}

variable "software_inventory" {
  description = "enable/disable software inventory (default is enabled)"
  default     = "1"
}

variable "vuln_db_path" {
  description = "the path to save the vuln database"
  default     = "/home/fleet"
}

variable "cpu_migrate" {
  description = "cpu units for migration task"
  default     = 1024
}

variable "mem_migrate" {
  description = "memory limit for migration task in MB"
  default     = 1024
}

variable "fleet_max_capacity" {
  description = "maximum number of fleet containers to run"
  default     = 5
}

variable "fleet_min_capacity" {
  description = "minimum number of fleet containers to run"
  default     = 1
}

variable "memory_tracking_target_value" {
  description = "target memory utilization for target tracking policy (default 80%)"
  default     = 80
}

variable "cpu_tracking_target_value" {
  description = "target cpu utilization for target tracking policy (default 60%)"
  default     = 60
}