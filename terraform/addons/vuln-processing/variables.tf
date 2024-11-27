variable "customer_prefix" {
  type        = string
  description = "customer prefix to use to namespace all resources"
  default     = "fleet"
}

variable "ecs_cluster" {
  type        = string
  description = "The ARN of the ECS cluster to use"
  nullable    = false
}

variable "vpc_id" {
  type    = string
  default = null
}

variable "fleet_config" {
  type = object({
    vuln_processing_schedule_expression  = optional(string, "rate(1 hour)")
    vuln_data_stream_schedule_expression = optional(string, "rate(24 hours)")
    vuln_database_path                   = optional(string, "/home/fleet/vuln_data")
    vuln_processing_mem                  = optional(number, 4096)
    vuln_processing_cpu                  = optional(number, 2048)
    vuln_data_stream_mem                 = optional(number, 1024)
    vuln_data_stream_cpu                 = optional(number, 512)
    image                                = optional(string, "fleetdm/fleet:v4.60.0")
    family                               = optional(string, "fleet-vuln-processing")
    sidecars                             = optional(list(any), [])
    extra_environment_variables          = optional(map(string), {})
    extra_iam_policies                   = optional(list(string), [])
    extra_execution_iam_policies         = optional(list(string), [])
    extra_secrets                        = optional(map(string), {})
    iam_role_arn                         = optional(string, null)
    database = object({
      password_secret_arn = string
      user                = string
      database            = string
      address             = string
      rr_address          = optional(string, null)
    })
    awslogs = optional(object({
      name      = optional(string, null)
      region    = optional(string, null)
      create    = optional(bool, true)
      prefix    = optional(string, "fleet-vuln")
      retention = optional(number, 5)
      }), {
      name      = null
      region    = null
      prefix    = "fleet"
      retention = 5
    })
    networking = object({
      subnets         = list(string)
      security_groups = optional(list(string), null)
    })
    iam = optional(object({
      role = optional(object({
        name        = optional(string, "fleet-vuln-processing-role")
        policy_name = optional(string, "fleet-vuln-processing-iam-policy")
        }), {
        name        = "fleet-vuln-processing-role"
        policy_name = "fleet-vuln-processing-iam-policy"
      })
      execution = optional(object({
        name        = optional(string, "fleet-vuln-processing-execution-role")
        policy_name = optional(string, "fleet-vuln-processing-execution-role")
        }), {
        name        = "fleet-vuln-processing-execution-role"
        policy_name = "fleet-vuln-processing-iam-policy-execution"
      })
      }), {
      name = "fleet-vuln-processing-execution-role"
    })
  })
  default = {
    vuln_processing_schedule_expression  = "rate(1 hour)"
    vuln_data_stream_schedule_expression = "rate(24 hours)"
    vuln_database_path                   = "/home/fleet/vuln_data"
    vuln_processing_mem                  = 4096
    vuln_processing_cpu                  = 2048
    vuln_data_stream_mem                 = 1024
    vuln_data_stream_cpu                 = 512
    image                                = "fleetdm/fleet:v4.60.0"
    family                               = "fleet-vuln-processing"
    sidecars                             = []
    extra_environment_variables          = {}
    extra_iam_policies                   = []
    extra_execution_iam_policies         = []
    extra_secrets                        = {}
    iam_role_arn                         = null
    database = {
      password_secret_arn = null
      user                = null
      database            = null
      address             = null
      rr_address          = null
    }
    awslogs = {
      name      = null
      region    = null
      create    = true
      prefix    = "fleet-vuln"
      retention = 5
    }
    networking = {
      subnets         = null
      security_groups = null
    }
    iam = {
      role = {
        name        = "fleet-vuln-processing-role"
        policy_name = "fleet-vuln-processing-iam-policy"
      }
      execution = {
        name        = "fleet-vuln-processing-execution-role"
        policy_name = "fleet-vuln-processing-iam-policy-execution"
      }
    }
  }
  description = "The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified."
  nullable    = false
}

variable "efs_root_directory" {
  default = "/"
}
