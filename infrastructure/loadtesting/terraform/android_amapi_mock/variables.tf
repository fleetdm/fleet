variable "tag" {
  description = "The git branch/tag to build the android-amapi-mock image from. Must be Docker-tag-safe (no forward slashes)."
  type        = string

  validation {
    condition     = can(regex("^[0-9A-Za-z_.-]+$", var.tag)) && length(var.tag) <= 128
    error_message = "var.tag must be a non-empty Docker-tag-safe string (letters, digits, '.', '_', '-' only, max 128 chars). Replace slashes with dashes."
  }
}

variable "enable_google_forwarding" {
  description = "Enable forwarding real device requests to Google AMAPI using credentials from the shared secret"
  type        = bool
  default     = false
}
