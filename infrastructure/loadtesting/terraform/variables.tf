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

variable "redis_instance_type" {
  description = "the redis instance type to use in loadtesting. default is cache.m6g.large"
  type        = string
  default     = "cache.m6g.large"
}

variable "mysql_max_open_conns" {
  description = "Max open MySQL connections per Fleet container, applied to both the writer and read-replica pools. Worst-case connections on a single Aurora instance is roughly fleet_containers * mysql_max_open_conns, and double that during a failover when reader connections shift onto the writer; this must stay under the instance's max_connections (verify with SELECT @@max_connections). Default 20 is safe for the R-class instances used at 25k+ hosts; lower it for smaller T-class instances. See https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/AuroraMySQL.Managing.Performance.html"
  type        = number
  default     = 20
}
