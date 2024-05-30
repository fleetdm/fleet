variable "fleet_iam_role_arn" {
  type        = string
  description = "The ARN of the IAM role that will be assuming into the IAM role defined in this module to gain permissions required to write to the Kinesis Data Stream(s)."
}

variable "sts_external_id" {
  type        = string
  description = "Optional unique identifier that can be used by the principal assuming the role to assert its identity."
  default     = ""
}

variable "log_destinations" {
  description = "A map of configurations for Kinesis data streams."
  type = map(object({
    name                = string
    shard_count         = number
    stream_mode         = string
    retention_period    = number
    shard_level_metrics = list(string)
  }))
  default = {
    results = {
      name                = "osquery_results"
      shard_count         = 0
      stream_mode         = "ON_DEMAND"
      retention_period    = 24
      shard_level_metrics = []
    },
    status = {
      name                = "osquery_status"
      shard_count         = 0
      stream_mode         = "ON_DEMAND"
      retention_period    = 24
      shard_level_metrics = []
    },
    audit = {
      name                = "fleet_audit"
      shard_count         = 0
      stream_mode         = "ON_DEMAND"
      retention_period    = 24
      shard_level_metrics = []
    }
  }
}