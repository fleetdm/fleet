variable "name" {
  description = "Name for the app and database cluster."
  type        = string
  default     = "fleet"
}

variable "app_region" {
  description = "App Platform region slug, such as nyc, sfo, ams, or fra."
  type        = string
  default     = "nyc"
}

variable "db_region" {
  description = "Database region slug. Choose the datacenter that matches app_region, such as nyc3 for nyc."
  type        = string
  default     = "nyc3"
}

variable "domain" {
  description = "Optional custom domain, such as fleet.example.com. Leave empty to use the free ondigitalocean.app address."
  type        = string
  default     = ""
}

variable "fleet_version" {
  description = "Fleet image tag, such as v4.95.0. Change it and apply to upgrade."
  type        = string
  default     = "v4.95.0"
}

variable "fleet_license_key" {
  description = "Optional Fleet Premium license key."
  type        = string
  default     = ""
  sensitive   = true
}

variable "fleet_instance_size" {
  description = "App Platform size slug for the Fleet server."
  type        = string
  default     = "apps-s-1vcpu-1gb-fixed"
}

variable "fleet_instance_count" {
  description = "Number of Fleet server instances."
  type        = number
  default     = 1
}

variable "mysql_size" {
  description = "Managed MySQL node size slug."
  type        = string
  default     = "db-s-1vcpu-2gb"
}

variable "mysql_node_count" {
  description = "Managed MySQL nodes. Use 2 or more for a standby node."
  type        = number
  default     = 1
}

variable "installers_bucket" {
  description = "Optional Spaces bucket name for software installers. Leave empty to skip."
  type        = string
  default     = ""
}

variable "installers_region" {
  description = "Region slug of the Spaces bucket, such as nyc3."
  type        = string
  default     = "nyc3"
}

variable "installers_access_key_id" {
  description = "Spaces access key ID for the bucket."
  type        = string
  default     = ""
  sensitive   = true
}

variable "installers_secret_access_key" {
  description = "Spaces secret key for the bucket."
  type        = string
  default     = ""
  sensitive   = true
}
