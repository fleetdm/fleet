variable "apn_secret_name" {
  default  = "fleet-apn"
  nullable = false
  type     = string
}

variable "scep_secret_name" {
  default  = "fleet-scep"
  nullable = false
  type     = string
}

variable "dep_secret_name" {
  default  = "fleet-dep"
  nullable = false
  type     = string
}

variable "public_domain_name" {
  nullable = false
  type     = string
}
