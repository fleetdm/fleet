variable "tag" {
  description = "The tag to deploy. This would be the same as the branch name"
  default     = "v4.76.1"
}

variable "fleet_task_count" {
  description = "The total number (max) that ECS can scale Fleet containers up to"
  type        = number
  default     = 5

  validation {
    condition     = var.fleet_task_count >= 0
    error_message = "var.fleet_task_count must be greater than or equal to 0."
  }
}

variable "fleet_task_memory" {
  description = "The memory configuration for Fleet containers"
  type        = number
  default     = 4096
}

variable "fleet_task_cpu" {
  description = "The CPU configuration for Fleet containers"
  type        = number
  default     = 512
}

variable "database_instance_size" {
  description = "The instance size for Aurora database instances"
  type        = string
  default     = "db.t4g.medium"
}

variable "database_instance_count" {
  description = "The number of Aurora database instances"
  type        = number
  default     = 2

  validation {
    condition     = var.database_instance_count >= 1
    error_message = "var.database_instance_count must be greater than or equal to 1."
  }
}

variable "redis_instance_size" {
  description = "The instance size for Elasticache nodes"
  type        = string
  default     = "cache.t4g.micro"
}

variable "redis_instance_count" {
  description = "The number of Elasticache nodes"
  type        = number
  default     = 3

  validation {
    condition     = var.redis_instance_count >= 1
    error_message = "var.redis_instance_count must be greater than or equal to 1."
  }
}

# The following three variables let load tests optionally run against Valkey instead of
# Redis. Defaults preserve the existing Redis 7.1 behavior. To provision Valkey, set all
# three to a matching set, e.g.:
#   -var=redis_engine=valkey -var=redis_engine_version=8.0 -var=redis_parameter_group_family=valkey8
variable "redis_engine" {
  description = "The Elasticache engine to use: \"redis\" or \"valkey\"."
  type        = string
  default     = "redis"

  validation {
    condition     = contains(["redis", "valkey"], var.redis_engine)
    error_message = "var.redis_engine must be either \"redis\" or \"valkey\"."
  }
}

variable "redis_engine_version" {
  description = "The Elasticache engine version (e.g. \"7.1\" for Redis, \"8.0\" for Valkey)."
  type        = string
  default     = "7.1"
}

variable "redis_parameter_group_family" {
  description = "The Elasticache parameter group family (e.g. \"redis7\", \"valkey7\", \"valkey8\"). Must match the engine and version."
  type        = string
  default     = "redis7"
}

variable "enable_otel" {
  description = "Enable OpenTelemetry tracing with SigNoz instead of Elastic APM"
  type        = bool
  default     = false
}
