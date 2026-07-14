variable "tag" {
  description = "The git branch/tag to build the android-amapi-mock image from"
  type        = string
}

variable "enable_google_forwarding" {
  description = "Enable forwarding real device requests to Google AMAPI using credentials from the shared secret"
  type        = bool
  default     = false
}
