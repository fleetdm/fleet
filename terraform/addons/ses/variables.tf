variable "name" {
  type        = string
  default     = ""
  description = "Name  (e.g. `app` or `cluster`)."
}

variable "environment" {
  type        = string
  default     = ""
  description = "Environment (e.g. `prod`, `dev`, `staging`)."
}

variable "repository" {
  type        = string
  default     = "https://github.com/clouddrove/terraform-aws-ses"
  description = "Terraform current module repo"
}

variable "label_order" {
  type        = list(any)
  default     = []
  description = "Label order, e.g. `name`,`application`."
}

variable "managedby" {
  type        = string
  default     = "hello@clouddrove.com"
  description = "ManagedBy, eg 'CloudDrove'"
}

variable "domain" {
  type        = string
  description = "Domain to use for SES."
}

variable "txt_type" {
  type        = string
  default     = "TXT"
  description = "Txt type for Record Set."
}

variable "cname_type" {
  type        = string
  default     = "CNAME"
  description = "CNAME type for Record Set."
}
