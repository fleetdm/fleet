variable "apn_secret_name" {
  default  = "fleet-apn"
  nullable = true
  type     = string
}

variable "scep_secret_name" {
  default  = "fleet-scep"
  nullable = false
  type     = string
}

variable "abm_secret_name" {
  default  = "fleet-abm"
  nullable = true
  type     = string
}

variable "enable_windows_mdm" {
  default  = false
  nullable = false
  type     = bool
}

variable "enable_apple_mdm" {
  default  = true
  nullable = false
  type     = bool
}
