variable "vpc_config" {
  type = object({
    vpc_id = string
    networking = object({
      subnets = list(string)
    })
  })
}


variable "rds_config" {
  type = object({
    name                            = string
    engine_version                  = string
    instance_class                  = string
    subnets                         = list(string)
    allowed_security_groups         = list(string)
    allowed_cidr_blocks             = list(string)
    apply_immediately               = bool
    monitoring_interval             = number
    db_parameter_group_name         = string
    db_cluster_parameter_group_name = string
    enabled_cloudwatch_logs_exports = list(string)
    master_username                 = optional(string, "fleet")
  })
  default = {
    name                            = "fleet"
    engine_version                  = "8.0.mysql_aurora.3.02.0"
    instance_class                  = "db.t4g.large"
    subnets                         = []
    allowed_security_groups         = []
    allowed_cidr_blocks             = []
    apply_immediately               = true
    monitoring_interval             = 10
    db_parameter_group_name         = null
    db_cluster_parameter_group_name = null
    enabled_cloudwatch_logs_exports = ["postgresql"]
    master_username                 = "fleet"
  }
  description = "The config for the terraform-aws-modules/rds-aurora/aws module"
  nullable    = false
}

variable "redis_config" {
  type = object({
    replication_group_id       = optional(string, "fleet")
    allowed_security_group_ids = optional(list(string), [])
    subnets                    = list(string)
    availability_zones         = list(string)
    cluster_size               = optional(number, 3)
    instance_type              = optional(string, "cache.m5.large")
    apply_immediately          = optional(bool, true)
    automatic_failover_enabled = optional(bool, false)
    engine_version             = optional(string, "6.x")
    family                     = optional(string, "redis")
    at_rest_encryption_enabled = optional(bool, true)
    transit_encryption_enabled = optional(bool, true)
    parameter = optional(list(object({
      name  = string
      value = string
    })), [])
  })
  default = {
    replication_group_id       = "fleet"
    allowed_security_group_ids = []
    subnets                    = null
    availability_zones         = null
    cluster_size               = 3
    instance_type              = "cache.m5.large"
    apply_immediately          = true
    automatic_failover_enabled = false
    engine_version             = "6.x"
    family                     = "redis"
    at_rest_encryption_enabled = true
    transit_encryption_enabled = true
    parameter                  = []
  }
}
