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
  type = string
  default = null
}

variable "alb_target_group_arn_suffix" {
  type = string
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


