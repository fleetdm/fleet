variable "git_tag_branch" {
  description = "The tag or git branch to use to build loadtest containers."
  type        = string
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

variable "stable_host_identities" {
  description = "Run each container as its own service with its own --seed, so a restarted or replaced container comes back as the same hosts instead of enrolling new ones"
  type        = bool
  default     = false
}

variable "task_size" {
  type = object({
    cpu    = optional(number, 4096)
    memory = optional(number, 8192)
  })

  default = {
    cpu    = 4096
    memory = 8192
  }
}
