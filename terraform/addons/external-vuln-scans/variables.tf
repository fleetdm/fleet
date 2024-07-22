variable "ecs_cluster" {
  description = "The ecs cluster module that is created by the byo-db module"
}

variable "fleet_config" {
  description = "The root Fleet config object"
  type        = any
}

variable "awslogs_config" {
  type = object({
    group  = string
    region = string
    prefix = string
  })
}

variable "subnets" {
  type     = list(string)
  nullable = false
}

variable "security_groups" {
  type     = list(string)
  nullable = false
}


variable "customer_prefix" {
  type    = string
  default = "fleet"
}

variable "fleet_s3_software_installers_config" {
  type        = map(string)
  description = "use the output of the byo-vpc module with the same name"
}

variable "fleet_server_private_key_secret_arn" {
  type        = string
  description = "The ARN of the secret that stores the Fleet private key"
}

variable "execution_iam_role_arn" {
  description = "The ARN of the fleet execution role, this is necessary to pass role from ecs events"
}

variable "task_role_arn" {
  description = "The ARN of the fleet task role, this is necessary to pass role from ecs events"
}

variable "vuln_processing_task_memory" {
  // note must conform to FARGATE breakpoints https://docs.aws.amazon.com/AmazonECS/latest/userguide/fargate-task-defs.html
  default     = 4096
  description = "The amount of memory to dedicate to the vuln processing task including sidecars"
}

variable "vuln_processing_task_cpu" {
  // note must conform to FARGETE breakpoints https://docs.aws.amazon.com/AmazonECS/latest/userguide/fargate-task-defs.html
  default     = 1024
  description = "The amount of CPU to dedicate to the vuln processing task including sidecars"
}


variable "vuln_processing_memory" {
  // note must conform to FARGATE breakpoints https://docs.aws.amazon.com/AmazonECS/latest/userguide/fargate-task-defs.html
  default     = 4096
  description = "The amount of memory to dedicate to the vuln processing command"
}

variable "vuln_processing_cpu" {
  // note must conform to FARGETE breakpoints https://docs.aws.amazon.com/AmazonECS/latest/userguide/fargate-task-defs.html
  default     = 1024
  description = "The amount of CPU to dedicate to the vuln processing command"
}

