variable "cookie_max_age" {
  type    = string
  default = "1h"
}

variable "alb_target_group_arn" {
  type = string
}

variable "alb_access_logs" {
  type    = map(string)
  default = {}
}

# variable "public_alb_security_group_id" {
#   type = string
# }

variable "idp_metadata_url" {
  type = string
}

variable "customer_prefix" {
  type        = string
  description = "customer prefix to use to namespace all resources"
  default     = "fleet"
}

variable "ecs_cluster" {
  type = string
}

variable "ecs_execution_iam_role_arn" {
  type = string
}

variable "ecs_iam_role_arn" {
  type = string
}

variable "proxy_containers" {
  type    = number
  default = 1
}

variable "logging_options" {
  type = object({
    awslogs-group         = string
    awslogs-region        = string
    awslogs-stream-prefix = string
  })
}

variable "saml_auth_proxy_image" {
  type    = string
  default = "itzg/saml-auth-proxy:1.12.0@sha256:ddff17caa00c1aad64d6c7b2e1d5eb93d97321c34d8ad12a25cfd8ce203db723"
}

variable "security_groups" {
  type     = list(string)
  nullable = false
}

variable "base_url" {
  type = string
}

variable "subnets" {
  type     = list(string)
  nullable = false
}

variable "vpc_id" {
  type     = string
  nullable = false
}
