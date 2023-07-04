variable "osquery_carve_s3_bucket" {
  type = object({
    name         = optional(string, "fleet-osquery-results-archive")
    expires_days = optional(number, 1)
  })
  default = {
    name         = "fleet-osquery-results-archive"
    expires_days = 1
  }
}
