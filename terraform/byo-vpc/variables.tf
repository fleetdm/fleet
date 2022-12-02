variable "vpc_id" {
  type     = string
  nullable = false
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
  }
  description = "The config for the terraform-aws-modules/rds-aurora/aws module"
  nullable    = false
}
