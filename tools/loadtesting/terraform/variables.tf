variable "tag" {
  description = "The tag to deploy. This would be the same as the branch name"
}

variable "fleet_config" {
  description = "The configuration to use for fleet itself, gets translated as environment variables"
  type        = map(string)
  default     = {}
}
