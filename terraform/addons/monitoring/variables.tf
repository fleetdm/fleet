variable "customer_prefix" {
  type    = string
  default = "fleet"
}

variable "fleet_ecs_service_name" {
  type    = string
  default = null
}

variable "fleet_min_containers" {
  type    = number
  default = 1
}

variable "alb_name" {
  type    = string
  default = null
}

variable "alb_target_group_name" {
  type    = string
  default = null
}

variable "alb_target_group_arn_suffix" {
  type    = string
  default = null
}

variable "alb_arn_suffix" {
  type    = string
  default = null
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
