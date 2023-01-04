## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.39.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_appautoscaling_policy.ecs_policy_cpu](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/appautoscaling_policy) | resource |
| [aws_appautoscaling_policy.ecs_policy_memory](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/appautoscaling_policy) | resource |
| [aws_appautoscaling_target.ecs_target](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/appautoscaling_target) | resource |
| [aws_cloudwatch_log_group.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_log_group) | resource |
| [aws_ecs_service.fleet](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_service) | resource |
| [aws_ecs_task_definition.backend](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_task_definition) | resource |
| [aws_iam_policy.execution](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.execution](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.execution](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.role_attachment](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_security_group.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |
| [aws_iam_policy_document.assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.fleet](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.fleet-execution](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster) | The name of the ECS cluster to use | `string` | n/a | yes |
| <a name="input_fleet_config"></a> [fleet\_config](#input\_fleet\_config) | The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified. | <pre>object({<br>    mem                         = optional(number, 512)<br>    cpu                         = optional(number, 256)<br>    image                       = optional(string, "fleetdm/fleet:v4.22.1")<br>    extra_environment_variables = optional(map(string), {})<br>    extra_secrets               = optional(map(string), {})<br>    security_groups             = optional(list(string), null)<br>    iam_role_arn                = optional(string, null)<br>    database = object({<br>      password_secret_arn = string<br>      user                = string<br>      database            = string<br>      address             = string<br>      rr_address          = optional(string, null)<br>    })<br>    redis = object({<br>      address = string<br>      use_tls = optional(bool, true)<br>    })<br>    awslogs = optional(object({<br>      name      = optional(string, null)<br>      region    = optional(string, null)<br>      prefix    = optional(string, "fleet")<br>      retention = optional(number, 5)<br>      }), {<br>      name      = null<br>      region    = null<br>      prefix    = "fleet"<br>      retention = 5<br>    })<br>    loadbalancer = object({<br>      arn = string<br>    })<br>    networking = object({<br>      subnets         = list(string)<br>      security_groups = optional(list(string), null)<br>    })<br>    autoscaling = optional(object({<br>      max_capacity                 = optional(number, 5)<br>      min_capacity                 = optional(number, 1)<br>      memory_tracking_target_value = optional(number, 80)<br>      cpu_tracking_target_value    = optional(number, 80)<br>      }), {<br>      max_capacity                 = 5<br>      min_capacity                 = 1<br>      memory_tracking_target_value = 80<br>      cpu_tracking_target_value    = 80<br>    })<br>  })</pre> | <pre>{<br>  "autoscaling": {<br>    "cpu_tracking_target_value": 80,<br>    "max_capacity": 5,<br>    "memory_tracking_target_value": 80,<br>    "min_capacity": 1<br>  },<br>  "awslogs": {<br>    "name": null,<br>    "prefix": "fleet",<br>    "region": null,<br>    "retention": 5<br>  },<br>  "cpu": 256,<br>  "database": {<br>    "address": null,<br>    "database": null,<br>    "password_secret_arn": null,<br>    "rr_address": null,<br>    "user": null<br>  },<br>  "extra_environment_variables": {},<br>  "extra_secrets": {},<br>  "iam_role_arn": null,<br>  "image": "fleetdm/fleet:v4.22.1",<br>  "loadbalancer": {<br>    "arn": null<br>  },<br>  "mem": 512,<br>  "networking": {<br>    "security_groups": null,<br>    "subnets": null<br>  },<br>  "redis": {<br>    "address": null,<br>    "use_tls": true<br>  },<br>  "security_groups": null<br>}</pre> | no |
| <a name="input_migration_config"></a> [migration\_config](#input\_migration\_config) | The configuration object for Fleet's migration task. | <pre>object({<br>    mem = number<br>    cpu = number<br>  })</pre> | <pre>{<br>  "cpu": 1024,<br>  "mem": 2048<br>}</pre> | no |
| <a name="input_vpc_id"></a> [vpc\_id](#input\_vpc\_id) | n/a | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_security_groups"></a> [security\_groups](#output\_security\_groups) | n/a |
