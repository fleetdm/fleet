variable "tag" {
  description = "The git branch/tag to build the android-amapi-mock image from. Must be Docker-tag-safe (no forward slashes)."
  type        = string

  validation {
    condition     = !strcontains(var.tag, "/")
    error_message = "var.tag cannot contain forward slashes — Docker image tags do not allow them. Replace slashes with dashes (e.g. \"my-feature\" instead of \"user/feature\")."
  }
}

variable "enable_google_forwarding" {
  description = "Enable forwarding real device requests to Google AMAPI using credentials from the shared secret"
  type        = bool
  default     = false
}
