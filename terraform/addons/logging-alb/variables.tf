variable "prefix" {
  type    = string
  default = "fleet"
}

variable "enable_athena" {
  type    = bool
  default = true
}

variable "s3_transition_days" {
  type    = number
  default = 30
}

variable "s3_expiration_days" {
  type    = number
  default = 90
}

variable "s3_newer_noncurrent_versions" {
  type    = number
  default = 5
}

variable "s3_noncurrent_version_expiration_days" {
  type    = number
  default = 30
}

variable "extra_kms_policies" {
  type    = list(any)
  default = []
}

variable "extra_s3_log_policies" {
  type    = list(any)
  default = []
}

variable "extra_s3_athena_policies" {
  type    = list(any)
  default = []
}
