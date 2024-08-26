variable "name" {}

variable "lb_arn" {}

variable "waf_type" {
  type    = string
  default = "blocklist"
}

variable "blocked_countries" {
  type    = list(string)
  default = ["BI", "BY", "CD", "CF", "CU", "IQ", "IR", "LB", "LY", "SD", "SO", "SS", "SY", "VE", "ZW", "RU"]
}

variable "blocked_addresses" {
  type    = list(string)
  default = []
}

variable "allowed_addresses" {
  type    = list(string)
  default = []
}
