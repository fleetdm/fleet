variable "customer_prefix" {
  type        = string
  description = "customer prefix to use to namespace all resources"
  default     = "fleet"
}

variable "ecs_cluster" {
  type     = string
}

variable "ecs_execution_iam_role_arn" {
  type     = string
}

variable "ecs_iam_role_arn" {
  type     = string
}

variable "extra_flags" {
  type     = list(string)
  default  = []
}

variable "loadtest_containers" {
  type     = number
  default  = 1
}

variable "logging_options" {
  type = object({
    awslogs-group         = string
    awslogs-region        = string
    awslogs-stream-prefix = string
  })
}

variable "osquery_perf_image" {
  type     = string
}

variable "security_groups" {
  type     = list(string)
  nullable = false
}

variable "server_url" {
  type     = string
}

variable "subnets" {
  type     = list(string)
  nullable = false
}
