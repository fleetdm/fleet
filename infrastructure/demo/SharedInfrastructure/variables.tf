variable "prefix" {}

variable "vpc_id" {}

variable "database_subnets" {
  type = list(string)
}

variable "allowed_cidr_blocks" {
  type = list(string)
}

variable "private_subnets" {
  type = list(string)
}

variable "allowed_security_groups" {
  type    = list(string)
  default = []
}

variable "eks_allowed_roles" {
  type    = list(any)
  default = []
}
