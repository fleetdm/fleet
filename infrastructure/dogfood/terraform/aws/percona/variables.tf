variable "zone_id" {
  description = "R53 Zone ID to host Percona in"
  type        = string
}

variable "domain_name" {
  description = "Domain name for Percona DNS"
  type        = string
}

variable "public_subnets" {
  description = "Public subnets for the Percona LB"
  type        = list(string)
}

variable "private_subnet" {
  description = "Private subnets for the Percona App instance"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}