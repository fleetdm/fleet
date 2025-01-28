variable "domain" {
  type        = string
  description = "Domain to use for SES."
}

variable "zone_id" {
  type        = string
  description = "Route53 Zone ID"
}

variable "extra_txt_records" {
  type        = list(string)
  description = "Extra TXT records that have to match the same name as the Fleet instance"
  default     = []
}
