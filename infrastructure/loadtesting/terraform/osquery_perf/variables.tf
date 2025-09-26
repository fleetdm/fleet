variable "tag" {
  description = "The tag to deploy. This would be the same as the branch name"
  type        = string
  default     = ""
}

variable "git_branch" {
  description = "The git branch to use to build loadtest containers.  Only needed if docker tag doesn't match the git branch"
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