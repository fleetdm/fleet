## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.14.0 |
| <a name="provider_random"></a> [random](#provider\_random) | 3.6.2 |

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
| [aws_iam_policy.software_installers](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.execution](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.execution](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.execution_extras](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.extras](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.role_attachment](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.software_installers](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_s3_bucket.software_installers](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket) | resource |
| [aws_s3_bucket_public_access_block.software_installers](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_public_access_block) | resource |
| [aws_s3_bucket_server_side_encryption_configuration.software_installers](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket_server_side_encryption_configuration) | resource |
| [aws_secretsmanager_secret.fleet_server_private_key](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_secretsmanager_secret_version.fleet_server_private_key](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret_version) | resource |
| [aws_security_group.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |
| [random_password.fleet_server_private_key](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password) | resource |
| [aws_iam_policy_document.assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.fleet](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.fleet-execution](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.software_installers](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster) | The name of the ECS cluster to use | `string` | n/a | yes |
| <a name="input_fleet_config"></a> [fleet\_config](#input\_fleet\_config) | The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified. | <pre>object({<br>    task_mem                     = optional(number, null)<br>    task_cpu                     = optional(number, null)<br>    mem                          = optional(number, 4096)<br>    cpu                          = optional(number, 512)<br>    pid_mode                     = optional(string, null)<br>    image                        = optional(string, "fleetdm/fleet:v4.54.1")<br>    family                       = optional(string, "fleet")<br>    sidecars                     = optional(list(any), [])<br>    depends_on                   = optional(list(any), [])<br>    mount_points                 = optional(list(any), [])<br>    volumes                      = optional(list(any), [])<br>    extra_environment_variables  = optional(map(string), {})<br>    extra_iam_policies           = optional(list(string), [])<br>    extra_execution_iam_policies = optional(list(string), [])<br>    extra_secrets                = optional(map(string), {})<br>    security_group_name          = optional(string, "fleet")<br>    iam_role_arn                 = optional(string, null)<br>    repository_credentials       = optional(string, "")<br>    private_key_secret_name      = optional(string, "fleet-server-private-key")<br>    service = optional(object({<br>      name = optional(string, "fleet")<br>      }), {<br>      name = "fleet"<br>    })<br>    database = object({<br>      password_secret_arn = string<br>      user                = string<br>      database            = string<br>      address             = string<br>      rr_address          = optional(string, null)<br>    })<br>    redis = object({<br>      address = string<br>      use_tls = optional(bool, true)<br>    })<br>    awslogs = optional(object({<br>      name      = optional(string, null)<br>      region    = optional(string, null)<br>      create    = optional(bool, true)<br>      prefix    = optional(string, "fleet")<br>      retention = optional(number, 5)<br>      }), {<br>      name      = null<br>      region    = null<br>      prefix    = "fleet"<br>      retention = 5<br>    })<br>    loadbalancer = object({<br>      arn = string<br>    })<br>    extra_load_balancers = optional(list(any), [])<br>    networking = object({<br>      subnets         = optional(list(string), null)<br>      security_groups = optional(list(string), null)<br>      ingress_sources = object({<br>        cidr_blocks      = optional(list(string), [])<br>        ipv6_cidr_blocks = optional(list(string), [])<br>        security_groups  = optional(list(string), [])<br>        prefix_list_ids  = optional(list(string), [])<br>      })<br>    })<br>    autoscaling = optional(object({<br>      max_capacity                 = optional(number, 5)<br>      min_capacity                 = optional(number, 1)<br>      memory_tracking_target_value = optional(number, 80)<br>      cpu_tracking_target_value    = optional(number, 80)<br>      }), {<br>      max_capacity                 = 5<br>      min_capacity                 = 1<br>      memory_tracking_target_value = 80<br>      cpu_tracking_target_value    = 80<br>    })<br>    iam = optional(object({<br>      role = optional(object({<br>        name        = optional(string, "fleet-role")<br>        policy_name = optional(string, "fleet-iam-policy")<br>        }), {<br>        name        = "fleet-role"<br>        policy_name = "fleet-iam-policy"<br>      })<br>      execution = optional(object({<br>        name        = optional(string, "fleet-execution-role")<br>        policy_name = optional(string, "fleet-execution-role")<br>        }), {<br>        name        = "fleet-execution-role"<br>        policy_name = "fleet-iam-policy-execution"<br>      })<br>      }), {<br>      name = "fleetdm-execution-role"<br>    })<br>    software_installers = optional(object({<br>      create_bucket    = optional(bool, true)<br>      bucket_name      = optional(string, null)<br>      bucket_prefix    = optional(string, "fleet-software-installers-")<br>      s3_object_prefix = optional(string, "")<br>      }), {<br>      create_bucket    = true<br>      bucket_name      = null<br>      bucket_prefix    = "fleet-software-installers-"<br>      s3_object_prefix = ""<br>    })<br>  })</pre> | <pre>{<br>  "autoscaling": {<br>    "cpu_tracking_target_value": 80,<br>    "max_capacity": 5,<br>    "memory_tracking_target_value": 80,<br>    "min_capacity": 1<br>  },<br>  "awslogs": {<br>    "create": true,<br>    "name": null,<br>    "prefix": "fleet",<br>    "region": null,<br>    "retention": 5<br>  },<br>  "cpu": 256,<br>  "database": {<br>    "address": null,<br>    "database": null,<br>    "password_secret_arn": null,<br>    "rr_address": null,<br>    "user": null<br>  },<br>  "depends_on": [],<br>  "extra_environment_variables": {},<br>  "extra_execution_iam_policies": [],<br>  "extra_iam_policies": [],<br>  "extra_load_balacners": [],<br>  "extra_secrets": {},<br>  "family": "fleet",<br>  "iam": {<br>    "execution": {<br>      "name": "fleet-execution-role",<br>      "policy_name": "fleet-iam-policy-execution"<br>    },<br>    "role": {<br>      "name": "fleet-role",<br>      "policy_name": "fleet-iam-policy"<br>    }<br>  },<br>  "iam_role_arn": null,<br>  "image": "fleetdm/fleet:v4.54.1",<br>  "loadbalancer": {<br>    "arn": null<br>  },<br>  "mem": 512,<br>  "mount_points": [],<br>  "networking": {<br>    "ingress_sources": {<br>      "cidr_blocks": [],<br>      "ipv6_cidr_blocks": [],<br>      "prefix_list_ids": [],<br>      "security_groups": []<br>    },<br>    "security_groups": null,<br>    "subnets": null<br>  },<br>  "pid_mode": null,<br>  "private_key_secret_name": "fleet-server-private-key",<br>  "redis": {<br>    "address": null,<br>    "use_tls": true<br>  },<br>  "repository_credentials": "",<br>  "security_group_name": "fleet",<br>  "security_groups": null,<br>  "service": {<br>    "name": "fleet"<br>  },<br>  "sidecars": [],<br>  "software_installers": {<br>    "bucket_name": null,<br>    "bucket_prefix": "fleet-software-installers-",<br>    "create_bucket": true,<br>    "s3_object_prefix": ""<br>  },<br>  "task_cpu": null,<br>  "task_mem": null,<br>  "volumes": []<br>}</pre> | no |
| <a name="input_migration_config"></a> [migration\_config](#input\_migration\_config) | The configuration object for Fleet's migration task. | <pre>object({<br>    mem = number<br>    cpu = number<br>  })</pre> | <pre>{<br>  "cpu": 1024,<br>  "mem": 2048<br>}</pre> | no |
| <a name="input_vpc_id"></a> [vpc\_id](#input\_vpc\_id) | n/a | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_appautoscaling_target"></a> [appautoscaling\_target](#output\_appautoscaling\_target) | n/a |
| <a name="output_execution_iam_role_arn"></a> [execution\_iam\_role\_arn](#output\_execution\_iam\_role\_arn) | n/a |
| <a name="output_fleet_config"></a> [fleet\_config](#output\_fleet\_config) | n/a |
| <a name="output_fleet_s3_software_installers_config"></a> [fleet\_s3\_software\_installers\_config](#output\_fleet\_s3\_software\_installers\_config) | n/a |
| <a name="output_fleet_server_private_key_secret_arn"></a> [fleet\_server\_private\_key\_secret\_arn](#output\_fleet\_server\_private\_key\_secret\_arn) | n/a |
| <a name="output_iam_role_arn"></a> [iam\_role\_arn](#output\_iam\_role\_arn) | n/a |
| <a name="output_logging_config"></a> [logging\_config](#output\_logging\_config) | n/a |
| <a name="output_non_circular"></a> [non\_circular](#output\_non\_circular) | n/a |
| <a name="output_service"></a> [service](#output\_service) | n/a |
| <a name="output_task_definition"></a> [task\_definition](#output\_task\_definition) | n/a |
