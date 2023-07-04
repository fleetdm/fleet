variable "ecs_cluster" {
  type     = string
  nullable = false
}

variable "task_definition" {
  type     = string
  nullable = false
}

variable "task_definition_revision" {
  type     = number
  nullable = false
}

variable "subnets" {
  type     = list(string)
  nullable = false
}

variable "security_groups" {
  type     = list(string)
  nullable = false
}
