variable "task_definition" {
  description = "The task definition resource that is created by the byo-ecs module"
}

variable "ecs_service" {
  description = "The ecs service resource that is created by the byo-ecs module"
}

variable "ecs_cluster" {
  description = "The ecs cluster module that is created by the byo-db module"
}
