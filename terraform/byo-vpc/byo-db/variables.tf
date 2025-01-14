variable "vpc_id" {
  type = string
}

variable "ecs_cluster" {
  type = object({
    autoscaling_capacity_providers = optional(any, {})
    cluster_configuration = optional(any, {
      execute_command_configuration = {
        logging = "OVERRIDE"
        log_configuration = {
          cloud_watch_log_group_name = "/aws/ecs/aws-ec2"
        }
      }
    })
    cluster_name = optional(string, "fleet")
    cluster_settings = optional(map(string), {
      "name" : "containerInsights",
      "value" : "enabled",
    })
    create                                = optional(bool, true)
    default_capacity_provider_use_fargate = optional(bool, true)
    fargate_capacity_providers = optional(any, {
      FARGATE = {
        default_capacity_provider_strategy = {
          weight = 100
        }
      }
      FARGATE_SPOT = {
        default_capacity_provider_strategy = {
          weight = 0
        }
      }
    })
    tags = optional(map(string))
  })
  default = {
    autoscaling_capacity_providers = {}
    cluster_configuration = {
      execute_command_configuration = {
        logging = "OVERRIDE"
        log_configuration = {
          cloud_watch_log_group_name = "/aws/ecs/aws-ec2"
        }
      }
    }
    cluster_name = "fleet"
    cluster_settings = {
      "name" : "containerInsights",
      "value" : "enabled",
    }
    create                                = true
    default_capacity_provider_use_fargate = true
    fargate_capacity_providers = {
      FARGATE = {
        default_capacity_provider_strategy = {
          weight = 100
        }
      }
      FARGATE_SPOT = {
        default_capacity_provider_strategy = {
          weight = 0
        }
      }
    }
    tags = {}
  }
  description = "The config for the terraform-aws-modules/ecs/aws module"
  nullable    = false
}


variable "fleet_config" {
  type = object({
    task_mem                     = optional(number, null)
    task_cpu                     = optional(number, null)
    mem                          = optional(number, 4096)
    cpu                          = optional(number, 512)
    pid_mode                     = optional(string, null)
    image                        = optional(string, "fleetdm/fleet:v4.62.1")
    family                       = optional(string, "fleet")
    sidecars                     = optional(list(any), [])
    depends_on                   = optional(list(any), [])
    mount_points                 = optional(list(any), [])
    volumes                      = optional(list(any), [])
    extra_environment_variables  = optional(map(string), {})
    extra_iam_policies           = optional(list(string), [])
    extra_execution_iam_policies = optional(list(string), [])
    extra_secrets                = optional(map(string), {})
    security_group_name          = optional(string, "fleet")
    iam_role_arn                 = optional(string, null)
    repository_credentials       = optional(string, "")
    private_key_secret_name      = optional(string, "fleet-server-private-key")
    service = optional(object({
      name = optional(string, "fleet")
      }), {
      name = "fleet"
    })
    database = optional(object({
      password_secret_arn = string
      user                = string
      database            = string
      address             = string
      rr_address          = optional(string, null)
      }), {
      password_secret_arn = null
      user                = null
      database            = null
      address             = null
      rr_address          = null
    })
    redis = optional(object({
      address = string
      use_tls = optional(bool, true)
      }), {
      address = null
      use_tls = true
    })
    awslogs = optional(object({
      name      = optional(string, null)
      region    = optional(string, null)
      create    = optional(bool, true)
      prefix    = optional(string, "fleet")
      retention = optional(number, 5)
      }), {
      name      = null
      region    = null
      prefix    = "fleet"
      retention = 5
    })
    loadbalancer = optional(object({
      arn = string
      }), {
      arn = null
    })
    extra_load_balancers = optional(list(any), [])
    networking = optional(object({
      subnets         = optional(list(string), null)
      security_groups = optional(list(string), null)
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
      }), {
      subnets         = null
      security_groups = null
      ingress_sources = {
        cidr_blocks      = []
        ipv6_cidr_blocks = []
        security_groups  = []
        prefix_list_ids  = []
      }
    })
    autoscaling = optional(object({
      max_capacity                 = optional(number, 5)
      min_capacity                 = optional(number, 1)
      memory_tracking_target_value = optional(number, 80)
      cpu_tracking_target_value    = optional(number, 80)
      }), {
      max_capacity                 = 5
      min_capacity                 = 1
      memory_tracking_target_value = 80
      cpu_tracking_target_value    = 80
    })
    iam = optional(object({
      role = optional(object({
        name        = optional(string, "fleet-role")
        policy_name = optional(string, "fleet-iam-policy")
        }), {
        name        = "fleet-role"
        policy_name = "fleet-iam-policy"
      })
      execution = optional(object({
        name        = optional(string, "fleet-execution-role")
        policy_name = optional(string, "fleet-execution-role")
        }), {
        name        = "fleet-execution-role"
        policy_name = "fleet-iam-policy-execution"
      })
      }), {
      name = "fleetdm-execution-role"
    })
    software_installers = optional(object({
      create_bucket    = optional(bool, true)
      bucket_name      = optional(string, null)
      bucket_prefix    = optional(string, "fleet-software-installers-")
      s3_object_prefix = optional(string, "")
      }), {
      create_bucket    = true
      bucket_name      = null
      bucket_prefix    = "fleet-software-installers-"
      s3_object_prefix = ""
    })
  })
  default = {
    task_mem                     = null
    task_cpu                     = null
    mem                          = 512
    cpu                          = 256
    pid_mode                     = null
    image                        = "fleetdm/fleet:v4.62.1"
    family                       = "fleet"
    sidecars                     = []
    depends_on                   = []
    volumes                      = []
    mount_points                 = []
    extra_environment_variables  = {}
    extra_iam_policies           = []
    extra_execution_iam_policies = []
    extra_secrets                = {}
    security_groups              = null
    security_group_name          = "fleet"
    iam_role_arn                 = null
    repository_credentials       = ""
    private_key_secret_name      = "fleet-server-private-key"
    service = {
      name = "fleet"
    }
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
      create    = true
      prefix    = "fleet"
      retention = 5
    }
    loadbalancer = {
      arn = null
    }
    extra_load_balancers = []
    networking = {
      subnets         = null
      security_groups = null
      ingress_sources = {
        cidr_blocks      = []
        ipv6_cidr_blocks = []
        security_groups  = []
        prefix_list_ids  = []
      }
    }
    autoscaling = {
      max_capacity                 = 5
      min_capacity                 = 1
      memory_tracking_target_value = 80
      cpu_tracking_target_value    = 80
    }
    iam = {
      role = {
        name        = "fleet-role"
        policy_name = "fleet-iam-policy"
      }
      execution = {
        name        = "fleet-execution-role"
        policy_name = "fleet-iam-policy-execution"
      }
    }
    software_installers = {
      create_bucket    = true
      bucket_name      = null
      bucket_prefix    = "fleet-software-installers-"
      s3_object_prefix = ""
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

variable "alb_config" {
  type = object({
    name                 = optional(string, "fleet")
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
