variable "tag" {
  description = "The tag to deploy. This would be the same as the branch name"
}

variable "git_branch" {
  description = "The git branch to use to build loadtest containers.  Only needed if docker tag doesn't match the git branch"
  type        = string
  default     = null
}

variable "fleet_config" {
  description = "The configuration to use for fleet itself, gets translated as environment variables"
  type        = map(string)
  default     = {}
}

variable "loadtest_containers" {
  description = "The number of containers to loadtest with"
  type        = number
  default     = 0
}

variable "fleet_containers" {
  description = "The number of containers running Fleet"
  type        = number
  default     = 10
}

variable "db_instance_type" {
  description = "The type of the loadtesting db instances.  Default is db.r6g.4xlarge."
  type        = string
  default     = "db.r6g.4xlarge"
}

variable "mysql_max_open_conns" {
  description = "Max open MySQL connections per Fleet container, applied to both the writer and read-replica pools. A single Aurora instance sees roughly fleet_containers * mysql_max_open_conns, up to 2x that on one instance during a failover or with no read replicas (when reader traffic falls back to the writer)."
  type        = number
  default     = 10

  validation {
    condition     = var.mysql_max_open_conns > 0
    error_message = "var.mysql_max_open_conns must be greater than 0 (0 means unlimited in database/sql, which can exhaust Aurora max_connections)."
  }
}

variable "redis_instance_type" {
  description = "the redis instance type to use in loadtesting. default is cache.m6g.large"
  type        = string
  default     = "cache.m6g.large"
}
