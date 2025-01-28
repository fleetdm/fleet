variable "awslogs_config" {
  type = object({
    group  = string
    region = string
    prefix = optional(string, "mdmproxy")
  })
}

variable "ecs_cluster" {
  type = string
}

variable "vpc_id" {
  type     = string
  nullable = false
}

variable "customer_prefix" {
  type    = string
  default = "fleet"
}

variable "config" {
  type = object({
    extra_execution_iam_policies = optional(list(string), [])
    mem                          = optional(number, 2048)
    cpu                          = optional(number, 1024)
    image                        = string
    desired_count                = optional(number, 1)
    repository_credentials       = optional(string, null)
    migrate_percentage           = number
    existing_hostname            = string
    existing_url                 = string
    fleet_url                    = string
    migrate_udids                = optional(list(string), [])
    auth_token                   = optional(string, "")
    iam = optional(object({
      execution = optional(object({
        name        = optional(string, "mdmproxy-execution-role")
        policy_name = optional(string, "mdmproxy-iam-policy-execution")
        }), {
        name        = "mdmproxy-execution-role"
        policy_name = "mdmproxy-iam-policy-execution"
      })
      }), {
      name        = "mdmproxy-execution-role"
      policy_name = "mdmproxy-iam-policy-execution"
    })
    networking = object({
      subnets             = list(string)
      security_groups     = optional(list(string), null)
      security_group_name = optional(string, "fleet-mdmproxy")
      ingress_sources = optional(object({
        cidr_blocks      = optional(list(string), [])
        ipv6_cidr_blocks = optional(list(string), [])
        security_groups  = optional(list(string), [])
        prefix_list_ids  = optional(list(string), [])
      }), {
        cidr_blocks      = []
        ipv6_cidr_blocks = []
        security_groups  = []
        prefix_list_ids  = []
      })
    })
  })
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
    idle_timeout         = optional(number, 905)
  })
}

