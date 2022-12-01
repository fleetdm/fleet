variable "ecs_cluster" {
  type        = string
  description = "The name of the ECS cluster to use"
  nullable    = false
}

variable "fleet_config" {
  type = object({
    mem                         = number
    cpu                         = number
    image                       = string
    extra_environment_variables = map(string)
    extra_secrets               = map(string)
    security_groups             = list(string)
    iam_role_arn                = string
    database = object({
      password_secret_arn = string
      user                = string
      database            = string
      address             = string
      rr_address          = string
    })
    redis = object({
      address = string
      use_tls = bool
    })
    awslogs = object({
      name      = string
      region    = string
      prefix    = string
      retention = number
    })
    loadbalancer = object({
      arn = string
    })
    networking = object({
      subnets         = list(string)
      security_groups = list(string)
    })
    autoscaling = object({
      max_capacity                 = number
      min_capacity                 = number
      memory_tracking_target_value = number
      cpu_tracking_target_value    = number
    })
  })
  default = {
    mem                         = 512
    cpu                         = 256
    image                       = "fleetdm/fleet:v4.22.1"
    extra_environment_variables = {}
    extra_secrets               = {}
    security_groups             = null
    iam_role_arn                = null
    database = {
      password_secret_arn = null
      user                = null
      database            = null
      address             = null
      rr_address          = null
    }
    redis = {
      address = null
      use_tls = true
    }
    awslogs = {
      name      = null
      region    = null
      prefix    = "fleet"
      retention = 5
    }
    loadbalancer = {
      arn = null
    }
    networking = {
      subnets         = null
      security_groups = null
    }
    autoscaling = {
      max_capacity                 = 5
      min_capacity                 = 1
      memory_tracking_target_value = 80
      cpu_tracking_target_value    = 80
    }
  }
  description = "The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified."
  nullable    = false
}

variable "migration_config" {
  type = object({
    mem = number
    cpu = number
  })
  default = {
    mem = 2048
    cpu = 1024
  }
  description = "The configuration object for Fleet's migration task."
  nullable    = false
}
