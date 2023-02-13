variable "vpc" {
  type = object({
    name                = optional(string, "fleet")
    cidr                = optional(string, "10.10.0.0/16")
    azs                 = optional(list(string), ["us-east-2a", "us-east-2b", "us-east-2c"])
    private_subnets     = optional(list(string), ["10.10.1.0/24", "10.10.2.0/24", "10.10.3.0/24"])
    public_subnets      = optional(list(string), ["10.10.11.0/24", "10.10.12.0/24", "10.10.13.0/24"])
    database_subnets    = optional(list(string), ["10.10.21.0/24", "10.10.22.0/24", "10.10.23.0/24"])
    elasticache_subnets = optional(list(string), ["10.10.31.0/24", "10.10.32.0/24", "10.10.33.0/24"])

    create_database_subnet_group          = optional(bool, false)
    create_database_subnet_route_table    = optional(bool, true)
    create_elasticache_subnet_group       = optional(bool, true)
    create_elasticache_subnet_route_table = optional(bool, true)
    enable_vpn_gateway                    = optional(bool, false)
    one_nat_gateway_per_az                = optional(bool, false)
    single_nat_gateway                    = optional(bool, true)
    enable_nat_gateway                    = optional(bool, true)
  })
  default = {
    name                = "fleet"
    cidr                = "10.10.0.0/16"
    azs                 = ["us-east-2a", "us-east-2b", "us-east-2c"]
    private_subnets     = ["10.10.1.0/24", "10.10.2.0/24", "10.10.3.0/24"]
    public_subnets      = ["10.10.11.0/24", "10.10.12.0/24", "10.10.13.0/24"]
    database_subnets    = ["10.10.21.0/24", "10.10.22.0/24", "10.10.23.0/24"]
    elasticache_subnets = ["10.10.31.0/24", "10.10.32.0/24", "10.10.33.0/24"]

    create_database_subnet_group          = false
    create_database_subnet_route_table    = true
    create_elasticache_subnet_group       = true
    create_elasticache_subnet_route_table = true
    enable_vpn_gateway                    = false
    one_nat_gateway_per_az                = false
    single_nat_gateway                    = true
    enable_nat_gateway                    = true
  }
}

variable "certificate_arn" {
  type = string
}

variable "rds_config" {
  type = object({
    name                            = optional(string, "fleet")
    engine_version                  = optional(string, "8.0.mysql_aurora.3.02.2")
    instance_class                  = optional(string, "db.t4g.large")
    subnets                         = optional(list(string), [])
    allowed_security_groups         = optional(list(string), [])
    allowed_cidr_blocks             = optional(list(string), [])
    apply_immediately               = optional(bool, true)
    monitoring_interval             = optional(number, 10)
    db_parameter_group_name         = optional(string)
    db_cluster_parameter_group_name = optional(string)
    enabled_cloudwatch_logs_exports = optional(list(string), [])
    master_username                 = optional(string, "fleet")
    snapshot_identifier             = optional(string)
  })
  default = {
    name                            = "fleet"
    engine_version                  = "8.0.mysql_aurora.3.02.2"
    instance_class                  = "db.t4g.large"
    subnets                         = []
    allowed_security_groups         = []
    allowed_cidr_blocks             = []
    apply_immediately               = true
    monitoring_interval             = 10
    db_parameter_group_name         = null
    db_cluster_parameter_group_name = null
    enabled_cloudwatch_logs_exports = []
    master_username                 = "fleet"
    snapshot_identifier             = null
  }
  description = "The config for the terraform-aws-modules/rds-aurora/aws module"
  nullable    = false
}

variable "redis_config" {
  type = object({
    name                          = optional(string, "fleet")
    replication_group_id          = optional(string)
    elasticache_subnet_group_name = optional(string)
    allowed_security_group_ids    = optional(list(string), [])
    subnets                       = optional(list(string))
    availability_zones            = optional(list(string))
    cluster_size                  = optional(number, 3)
    instance_type                 = optional(string, "cache.m5.large")
    apply_immediately             = optional(bool, true)
    automatic_failover_enabled    = optional(bool, false)
    engine_version                = optional(string, "6.x")
    family                        = optional(string, "redis6.x")
    at_rest_encryption_enabled    = optional(bool, true)
    transit_encryption_enabled    = optional(bool, true)
    parameter = optional(list(object({
      name  = string
      value = string
    })), [])
  })
  default = {
    name                          = "fleet"
    replication_group_id          = null
    elasticache_subnet_group_name = null
    allowed_security_group_ids    = []
    subnets                       = null
    availability_zones            = null
    cluster_size                  = 3
    instance_type                 = "cache.m5.large"
    apply_immediately             = true
    automatic_failover_enabled    = false
    engine_version                = "6.x"
    family                        = "redis6.x"
    at_rest_encryption_enabled    = true
    transit_encryption_enabled    = true
    parameter                     = []
  }
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
    mem                          = optional(number, 4096)
    cpu                          = optional(number, 512)
    image                        = optional(string, "fleetdm/fleet:v4.22.1")
    family                       = optional(string, "fleet")
    extra_environment_variables  = optional(map(string), {})
    extra_iam_policies           = optional(list(string), [])
    extra_execution_iam_policies = optional(list(string), [])
    extra_secrets                = optional(map(string), {})
    security_groups              = optional(list(string), null)
    security_group_name          = optional(string, "fleet")
    iam_role_arn                 = optional(string, null)
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
    networking = optional(object({
      subnets         = list(string)
      security_groups = optional(list(string), null)
      }), {
      subnets         = null
      security_groups = null
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
  })
  default = {
    mem                          = 512
    cpu                          = 256
    image                        = "fleetdm/fleet:v4.22.1"
    family                       = "fleet"
    extra_environment_variables  = {}
    extra_iam_policies           = []
    extra_execution_iam_policies = []
    extra_secrets                = {}
    security_groups              = null
    security_group_name          = "fleet"
    iam_role_arn                 = null
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
  }
  description = "The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified."
  nullable    = false
}

variable "migration_config" {
  type = object({
    mem = number
    cpu = number
  })
  description = "The configuration object for Fleet's migration task."
  nullable    = false
  default = {
    mem = 2048
    cpu = 1024
  }
}

variable "alb_config" {
  type = object({
    name            = optional(string, "fleet")
    security_groups = optional(list(string), [])
    access_logs     = optional(map(string), {})
  })
  default = {}
}
