variable "git_ref" {
  description = "The git branch or tag of fleetdm/fleet to build the driver and fleetctl from (usually the load test's tag)."
  type        = string
}

variable "desired_count" {
  description = "Driver tasks to run: 1, or 0 to stop it. Apply with 0 first and set the secrets' values (see README) before 1."
  type        = number
  default     = 1

  validation {
    condition     = contains([0, 1], var.desired_count)
    error_message = "Run one driver per load test (0 or 1): two would double the workload and fight over the GitOps repo."
  }
}

variable "fleet_url" {
  description = "Fleet's URL for the admin API. Default: the load test's internal load balancer."
  type        = string
  default     = ""
}

variable "gitops_bundle_key" {
  description = "Key of the private GitOps repo tarball in the driver's bucket (with its own apply.sh), for gitops.apply. Empty: no GitOps runs."
  type        = string
  default     = ""
}

variable "with_profile" {
  description = "Pass the private profile secret (the customer's measured rates). Without it, every spec runs at its public start rate."
  type        = bool
  default     = true
}

variable "notify" {
  description = "Create a webhook secret (Slack incoming webhook) the driver posts trips to."
  type        = bool
  default     = false
}

variable "extra_flags" {
  description = "More driver flags, e.g. [\"-only\", \"W-USER-01,C-INFRA-01\"]."
  type        = list(string)
  default     = []
}

variable "task_size" {
  description = "Fargate size. fleetctl gitops on a large repo needs the memory; the driver alone needs little."
  type = object({
    cpu    = optional(number, 512)
    memory = optional(number, 2048)
  })
  default = {}
}
