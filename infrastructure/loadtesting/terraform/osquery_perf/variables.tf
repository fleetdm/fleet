variable "git_tag_branch" {
  description = "The tag or git branch to use to build loadtest containers."
  type        = string
  default     = null
}

variable "loadtest_containers" {
  description = "Number of loadtest containers to deploy"
  type        = number
  default     = 1
}

variable "extra_flags" {
  description = "Comma delimited list (string) for passing extra flags to osquery-perf containers"
  type        = list(string)
  default     = ["--orbit_prob", "0.0"]
}
