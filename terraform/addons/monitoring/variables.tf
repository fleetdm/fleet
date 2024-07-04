variable "customer_prefix" {
  type    = string
  default = "fleet"
}

variable "fleet_ecs_service_name" {
  type    = string
  default = null
}

variable "albs" {
  type = list(object({
    name                    = string
    arn_suffix              = string
    target_group_name       = string
    target_group_arn_suffix = string
    min_containers          = optional(string, 1)
    ecs_service_name        = string
    alert_thresholds = optional(
      object({
        HTTPCode_ELB_5XX_Count = object({
          period    = number
          threshold = number
        })
        HTTPCode_Target_5XX_Count = object({
          period    = number
          threshold = number
        })
      }),
      {
        HTTPCode_ELB_5XX_Count = {
          period    = 120
          threshold = 0
        },
        HTTPCode_Target_5XX_Count = {
          period    = 120
          threshold = 0
        }
      }
    )
  }))
  default = []
}


variable "default_sns_topic_arns" {
  type    = list(string)
  default = []
}

variable "sns_topic_arns_map" {
  type    = map(list(string))
  default = {}
}

variable "mysql_cluster_members" {
  type    = list(string)
  default = []
}

variable "redis_cluster_members" {
  type    = list(string)
  default = []
}

variable "acm_certificate_arn" {
  type    = string
  default = null
}

variable "cron_monitoring" {
  type = object({
    mysql_host                 = string
    mysql_database             = string
    mysql_user                 = string
    mysql_password_secret_name = string
    vpc_id                     = string
    subnet_ids                 = list(string)
    rds_security_group_id      = string
    delay_tolerance            = string
    run_interval               = string
    log_retention_in_days      = optional(number, 7)
  })
  default = null
}
