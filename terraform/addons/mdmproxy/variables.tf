variable "awslogs_config" {
  type = object({
    group  = string
    region = string
    prefix = string
  })
}

variable "ecs_cluster" {
  type = string
}

variable "vpc_id" {
  type     = string
  nullable = false
}

variable "subnets" {
  type     = list(string)
  nullable = false
}

variable "security_groups" {
  type     = list(string)
  nullable = false
}

variable "customer_prefix" {
  type    = string
  default = "fleet"
}

variable "config" {
  type = any
}

variable "alb_config" {
  type = object({
    name                 = optional(string, "fleet-mdmproxy")
    subnets              = list(string)
    security_groups      = optional(list(string), [])
    access_logs          = optional(map(string), {})
    certificate_arn      = string
    allowed_cidrs        = optional(list(string), ["0.0.0.0/0"])
    allowed_ipv6_cidrs   = optional(list(string), ["::/0"])
    egress_cidrs         = optional(list(string), ["0.0.0.0/0"])
    egress_ipv6_cidrs    = optional(list(string), ["::/0"])
    extra_target_groups  = optional(any, [])
    https_listener_rules = optional(any, [])
    tls_policy           = optional(string, "ELBSecurityPolicy-TLS-1-2-2017-01")
    idle_timeout         = optional(number, 60)
  })
}

