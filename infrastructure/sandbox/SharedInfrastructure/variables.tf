variable "prefix" {}

variable "allowed_security_groups" {
  type    = list(string)
  default = []
}

variable "eks_allowed_roles" {
  type    = list(any)
  default = []
}

variable "vpc" {}
variable "base_domain" {}
variable "kms_key" {}
