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

variable "osquery_host_count" {
  default = 50
}

variable "loadtesting" {
  default = false
}

variable "vulnerabilities_path" {
  default = "/home/fleet"
}

variable "software_inventory" {
  default = "1"
}