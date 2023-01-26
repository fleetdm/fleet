## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.40.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_alb"></a> [alb](#module\_alb) | terraform-aws-modules/alb/aws | 8.2.1 |
| <a name="module_cluster"></a> [cluster](#module\_cluster) | terraform-aws-modules/ecs/aws | 4.1.2 |
| <a name="module_ecs"></a> [ecs](#module\_ecs) | ./byo-ecs | n/a |

## Resources

| Name | Type |
|------|------|
| [aws_security_group.alb](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alb_config"></a> [alb\_config](#input\_alb\_config) | n/a | <pre>object({<br>    name            = optional(string, "fleet")<br>    subnets         = list(string)<br>    security_groups = optional(list(string), [])<br>    access_logs     = optional(map(string), {})<br>    certificate_arn = string<br>  })</pre> | n/a | yes |
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster) | The config for the terraform-aws-modules/ecs/aws module | <pre>object({<br>    autoscaling_capacity_providers        = any<br>    cluster_configuration                 = any<br>    cluster_name                          = string<br>    cluster_settings                      = map(string)<br>    create                                = bool<br>    default_capacity_provider_use_fargate = bool<br>    fargate_capacity_providers            = any<br>    tags                                  = map(string)<br>  })</pre> | <pre>{<br>  "autoscaling_capacity_providers": {},<br>  "cluster_configuration": {<br>    "execute_command_configuration": {<br>      "log_configuration": {<br>        "cloud_watch_log_group_name": "/aws/ecs/aws-ec2"<br>      },<br>      "logging": "OVERRIDE"<br>    }<br>  },<br>  "cluster_name": "fleet",<br>  "cluster_settings": {<br>    "name": "containerInsights",<br>    "value": "enabled"<br>  },<br>  "create": true,<br>  "default_capacity_provider_use_fargate": true,<br>  "fargate_capacity_providers": {<br>    "FARGATE": {<br>      "default_capacity_provider_strategy": {<br>        "weight": 100<br>      }<br>    },<br>    "FARGATE_SPOT": {<br>      "default_capacity_provider_strategy": {<br>        "weight": 0<br>      }<br>    }<br>  },<br>  "tags": {}<br>}</pre> | no |
| <a name="input_fleet_config"></a> [fleet\_config](#input\_fleet\_config) | The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified. | <pre>object({<br>    mem                         = optional(number, 512)<br>    cpu                         = optional(number, 256)<br>    image                       = optional(string, "fleetdm/fleet:v4.22.1")<br>    family                      = optional(string, "fleet")<br>    extra_environment_variables = optional(map(string), {})<br>    extra_iam_policies          = optional(list(string), [])<br>    extra_secrets               = optional(map(string), {})<br>    security_groups             = optional(list(string), null)<br>    security_group_name         = optional(string, "fleet")<br>    iam_role_arn                = optional(string, null)<br>    service = optional(object({<br>      name = optional(string, "fleet")<br>      }), {<br>      name = "fleet"<br>    })<br>    database = object({<br>      password_secret_arn = string<br>      user                = string<br>      database            = string<br>      address             = string<br>      rr_address          = optional(string, null)<br>    })<br>    redis = object({<br>      address = string<br>      use_tls = optional(bool, true)<br>    })<br>    awslogs = optional(object({<br>      name      = optional(string, null)<br>      region    = optional(string, null)<br>      create    = optional(bool, true)<br>      prefix    = optional(string, "fleet")<br>      retention = optional(number, 5)<br>      }), {<br>      name      = null<br>      region    = null<br>      prefix    = "fleet"<br>      retention = 5<br>    })<br>    loadbalancer = object({<br>      arn = string<br>    })<br>    networking = object({<br>      subnets         = list(string)<br>      security_groups = optional(list(string), null)<br>    })<br>    autoscaling = optional(object({<br>      max_capacity                 = optional(number, 5)<br>      min_capacity                 = optional(number, 1)<br>      memory_tracking_target_value = optional(number, 80)<br>      cpu_tracking_target_value    = optional(number, 80)<br>      }), {<br>      max_capacity                 = 5<br>      min_capacity                 = 1<br>      memory_tracking_target_value = 80<br>      cpu_tracking_target_value    = 80<br>    })<br>    iam = optional(object({<br>      role = optional(object({<br>        name        = optional(string, "fleet-role")<br>        policy_name = optional(string, "fleet-iam-policy")<br>        }), {<br>        name        = "fleet-role"<br>        policy_name = "fleet-iam-policy"<br>      })<br>      execution = optional(object({<br>        name        = optional(string, "fleet-execution-role")<br>        policy_name = optional(string, "fleet-execution-role")<br>        }), {<br>        name        = "fleet-execution-role"<br>        policy_name = "fleet-iam-policy-execution"<br>      })<br>      }), {<br>      name = "fleetdm-execution-role"<br>    })<br>  })</pre> | <pre>{<br>  "autoscaling": {<br>    "cpu_tracking_target_value": 80,<br>    "max_capacity": 5,<br>    "memory_tracking_target_value": 80,<br>    "min_capacity": 1<br>  },<br>  "awslogs": {<br>    "create": true,<br>    "name": null,<br>    "prefix": "fleet",<br>    "region": null,<br>    "retention": 5<br>  },<br>  "cpu": 256,<br>  "database": {<br>    "address": null,<br>    "database": null,<br>    "password_secret_arn": null,<br>    "rr_address": null,<br>    "user": null<br>  },<br>  "extra_environment_variables": {},<br>  "extra_iam_policies": [],<br>  "extra_secrets": {},<br>  "family": "fleet",<br>  "iam": {<br>    "execution": {<br>      "name": "fleet-execution-role",<br>      "policy_name": "fleet-iam-policy-execution"<br>    },<br>    "role": {<br>      "name": "fleet-role",<br>      "policy_name": "fleet-iam-policy"<br>    }<br>  },<br>  "iam_role_arn": null,<br>  "image": "fleetdm/fleet:v4.22.1",<br>  "loadbalancer": {<br>    "arn": null<br>  },<br>  "mem": 512,<br>  "networking": {<br>    "security_groups": null,<br>    "subnets": null<br>  },<br>  "redis": {<br>    "address": null,<br>    "use_tls": true<br>  },<br>  "security_group_name": "fleet",<br>  "security_groups": null,<br>  "service": {<br>    "name": "fleet"<br>  }<br>}</pre> | no |
| <a name="input_migration_config"></a> [migration\_config](#input\_migration\_config) | The configuration object for Fleet's migration task. | <pre>object({<br>    mem = number<br>    cpu = number<br>  })</pre> | <pre>{<br>  "cpu": 1024,<br>  "mem": 2048<br>}</pre> | no |
| <a name="input_vpc_id"></a> [vpc\_id](#input\_vpc\_id) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_alb"></a> [alb](#output\_alb) | n/a |
| <a name="output_byo-ecs"></a> [byo-ecs](#output\_byo-ecs) | n/a |
