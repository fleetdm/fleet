This module provides a basic Fleet setup. This assumes that you bring nothing to the installation.
If you want to bring your own VPC/database/cache nodes/ECS cluster, then use one of the submodules provided.

The following is the module layout so you can navigate to the module that you want:

* Root module (use this to get a Fleet instance ASAP with minimal setup)
    * BYO-VPC (use this if you want to install Fleet inside an existing VPC)
        * BYO-database (use this if you want to use an existing database and cache node)
            * BYO-ECS (use this if you want to bring your own everything but Fleet ECS services)

# Migrating from existing Dogfood code
The below code describes how to migrate from existing Dogfood code

```hcl
moved {
  from = module.vpc
  to   = module.main.module.vpc
}

moved {
  from = module.aurora_mysql
  to = module.main.module.byo-vpc.module.rds
}

moved {
  from = aws_elasticache_replication_group.default
  to = module.main.module.byo-vpc.module.redis.aws_elasticache_replication_group.default
}
```

This focuses on the resources that are "heavy" or store data. Note that the ALB cannot be moved like this because Dogfood uses the `aws_alb` resource and the module uses the `aws_lb` resource. The resources are aliases of eachother, but Terraform can't recognize that.

# How to improve this module
If this module somehow doesn't fit your needs, feel free to contact us by
opening a ticket, or contacting your contact at Fleet. Our goal is to make this module
fit all needs within AWS, so we will try to find a solution so that this module fits your needs.

If you want to make the changes yourself, simply make a PR into main with your additions.
We would ask that you make sure that variables are defined as null if there is
no default that makes sense and that variable changes are reflected all the way up the stack.

# How to update this readme
Edit .header.md and run `terraform-docs markdown . > README.md`

## Requirements

No requirements.

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_byo-vpc"></a> [byo-vpc](#module\_byo-vpc) | ./byo-vpc | n/a |
| <a name="module_vpc"></a> [vpc](#module\_vpc) | terraform-aws-modules/vpc/aws | 3.18.1 |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_alb_config"></a> [alb\_config](#input\_alb\_config) | n/a | <pre>object({<br>    name            = optional(string, "fleet")<br>    security_groups = optional(list(string), [])<br>    access_logs     = optional(map(string), {})<br>  })</pre> | `{}` | no |
| <a name="input_certificate_arn"></a> [certificate\_arn](#input\_certificate\_arn) | n/a | `string` | n/a | yes |
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster) | The config for the terraform-aws-modules/ecs/aws module | <pre>object({<br>    autoscaling_capacity_providers = optional(any, {})<br>    cluster_configuration = optional(any, {<br>      execute_command_configuration = {<br>        logging = "OVERRIDE"<br>        log_configuration = {<br>          cloud_watch_log_group_name = "/aws/ecs/aws-ec2"<br>        }<br>      }<br>    })<br>    cluster_name = optional(string, "fleet")<br>    cluster_settings = optional(map(string), {<br>      "name" : "containerInsights",<br>      "value" : "enabled",<br>    })<br>    create                                = optional(bool, true)<br>    default_capacity_provider_use_fargate = optional(bool, true)<br>    fargate_capacity_providers = optional(any, {<br>      FARGATE = {<br>        default_capacity_provider_strategy = {<br>          weight = 100<br>        }<br>      }<br>      FARGATE_SPOT = {<br>        default_capacity_provider_strategy = {<br>          weight = 0<br>        }<br>      }<br>    })<br>    tags = optional(map(string))<br>  })</pre> | <pre>{<br>  "autoscaling_capacity_providers": {},<br>  "cluster_configuration": {<br>    "execute_command_configuration": {<br>      "log_configuration": {<br>        "cloud_watch_log_group_name": "/aws/ecs/aws-ec2"<br>      },<br>      "logging": "OVERRIDE"<br>    }<br>  },<br>  "cluster_name": "fleet",<br>  "cluster_settings": {<br>    "name": "containerInsights",<br>    "value": "enabled"<br>  },<br>  "create": true,<br>  "default_capacity_provider_use_fargate": true,<br>  "fargate_capacity_providers": {<br>    "FARGATE": {<br>      "default_capacity_provider_strategy": {<br>        "weight": 100<br>      }<br>    },<br>    "FARGATE_SPOT": {<br>      "default_capacity_provider_strategy": {<br>        "weight": 0<br>      }<br>    }<br>  },<br>  "tags": {}<br>}</pre> | no |
| <a name="input_fleet_config"></a> [fleet\_config](#input\_fleet\_config) | The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified. | <pre>object({<br>    mem                         = optional(number, 512)<br>    cpu                         = optional(number, 256)<br>    image                       = optional(string, "fleetdm/fleet:v4.22.1")<br>    family                      = optional(string, "fleet")<br>    extra_environment_variables = optional(map(string), {})<br>    extra_iam_policies          = optional(list(string), [])<br>    extra_secrets               = optional(map(string), {})<br>    security_groups             = optional(list(string), null)<br>    security_group_name         = optional(string, "fleet")<br>    iam_role_arn                = optional(string, null)<br>    service = optional(object({<br>      name = optional(string, "fleet")<br>      }), {<br>      name = "fleet"<br>    })<br>    database = optional(object({<br>      password_secret_arn = string<br>      user                = string<br>      database            = string<br>      address             = string<br>      rr_address          = optional(string, null)<br>      }), {<br>      password_secret_arn = null<br>      user                = null<br>      database            = null<br>      address             = null<br>      rr_address          = null<br>    })<br>    redis = optional(object({<br>      address = string<br>      use_tls = optional(bool, true)<br>      }), {<br>      address = null<br>      use_tls = true<br>    })<br>    awslogs = optional(object({<br>      name      = optional(string, null)<br>      region    = optional(string, null)<br>      create    = optional(bool, true)<br>      prefix    = optional(string, "fleet")<br>      retention = optional(number, 5)<br>      }), {<br>      name      = null<br>      region    = null<br>      prefix    = "fleet"<br>      retention = 5<br>    })<br>    loadbalancer = optional(object({<br>      arn = string<br>      }), {<br>      arn = null<br>    })<br>    networking = optional(object({<br>      subnets         = list(string)<br>      security_groups = optional(list(string), null)<br>      }), {<br>      subnets         = null<br>      security_groups = null<br>    })<br>    autoscaling = optional(object({<br>      max_capacity                 = optional(number, 5)<br>      min_capacity                 = optional(number, 1)<br>      memory_tracking_target_value = optional(number, 80)<br>      cpu_tracking_target_value    = optional(number, 80)<br>      }), {<br>      max_capacity                 = 5<br>      min_capacity                 = 1<br>      memory_tracking_target_value = 80<br>      cpu_tracking_target_value    = 80<br>    })<br>    iam = optional(object({<br>      role = optional(object({<br>        name        = optional(string, "fleet-role")<br>        policy_name = optional(string, "fleet-iam-policy")<br>        }), {<br>        name        = "fleet-role"<br>        policy_name = "fleet-iam-policy"<br>      })<br>      execution = optional(object({<br>        name        = optional(string, "fleet-execution-role")<br>        policy_name = optional(string, "fleet-execution-role")<br>        }), {<br>        name        = "fleet-execution-role"<br>        policy_name = "fleet-iam-policy-execution"<br>      })<br>      }), {<br>      name = "fleetdm-execution-role"<br>    })<br>  })</pre> | <pre>{<br>  "autoscaling": {<br>    "cpu_tracking_target_value": 80,<br>    "max_capacity": 5,<br>    "memory_tracking_target_value": 80,<br>    "min_capacity": 1<br>  },<br>  "awslogs": {<br>    "create": true,<br>    "name": null,<br>    "prefix": "fleet",<br>    "region": null,<br>    "retention": 5<br>  },<br>  "cpu": 256,<br>  "database": {<br>    "address": null,<br>    "database": null,<br>    "password_secret_arn": null,<br>    "rr_address": null,<br>    "user": null<br>  },<br>  "extra_environment_variables": {},<br>  "extra_iam_policies": [],<br>  "extra_secrets": {},<br>  "family": "fleet",<br>  "iam": {<br>    "execution": {<br>      "name": "fleet-execution-role",<br>      "policy_name": "fleet-iam-policy-execution"<br>    },<br>    "role": {<br>      "name": "fleet-role",<br>      "policy_name": "fleet-iam-policy"<br>    }<br>  },<br>  "iam_role_arn": null,<br>  "image": "fleetdm/fleet:v4.22.1",<br>  "loadbalancer": {<br>    "arn": null<br>  },<br>  "mem": 512,<br>  "networking": {<br>    "security_groups": null,<br>    "subnets": null<br>  },<br>  "redis": {<br>    "address": null,<br>    "use_tls": true<br>  },<br>  "security_group_name": "fleet",<br>  "security_groups": null,<br>  "service": {<br>    "name": "fleet"<br>  }<br>}</pre> | no |
| <a name="input_migration_config"></a> [migration\_config](#input\_migration\_config) | The configuration object for Fleet's migration task. | <pre>object({<br>    mem = number<br>    cpu = number<br>  })</pre> | <pre>{<br>  "cpu": 1024,<br>  "mem": 2048<br>}</pre> | no |
| <a name="input_rds_config"></a> [rds\_config](#input\_rds\_config) | The config for the terraform-aws-modules/rds-aurora/aws module | <pre>object({<br>    name                            = optional(string, "fleet")<br>    engine_version                  = optional(string, "8.0.mysql_aurora.3.02.2")<br>    instance_class                  = optional(string, "db.t4g.large")<br>    subnets                         = optional(list(string), [])<br>    allowed_security_groups         = optional(list(string), [])<br>    allowed_cidr_blocks             = optional(list(string), [])<br>    apply_immediately               = optional(bool, true)<br>    monitoring_interval             = optional(number, 10)<br>    db_parameter_group_name         = optional(string)<br>    db_cluster_parameter_group_name = optional(string)<br>    enabled_cloudwatch_logs_exports = optional(list(string), [])<br>    master_username                 = optional(string, "fleet")<br>  })</pre> | <pre>{<br>  "allowed_cidr_blocks": [],<br>  "allowed_security_groups": [],<br>  "apply_immediately": true,<br>  "db_cluster_parameter_group_name": null,<br>  "db_parameter_group_name": null,<br>  "enabled_cloudwatch_logs_exports": [],<br>  "engine_version": "8.0.mysql_aurora.3.02.2",<br>  "instance_class": "db.t4g.large",<br>  "master_username": "fleet",<br>  "monitoring_interval": 10,<br>  "name": "fleet",<br>  "subnets": []<br>}</pre> | no |
| <a name="input_redis_config"></a> [redis\_config](#input\_redis\_config) | n/a | <pre>object({<br>    name                          = optional(string, "fleet")<br>    replication_group_id          = optional(string)<br>    elasticache_subnet_group_name = optional(string)<br>    allowed_security_group_ids    = optional(list(string), [])<br>    subnets                       = optional(list(string))<br>    availability_zones            = optional(list(string))<br>    cluster_size                  = optional(number, 3)<br>    instance_type                 = optional(string, "cache.m5.large")<br>    apply_immediately             = optional(bool, true)<br>    automatic_failover_enabled    = optional(bool, false)<br>    engine_version                = optional(string, "6.x")<br>    family                        = optional(string, "redis6.x")<br>    at_rest_encryption_enabled    = optional(bool, true)<br>    transit_encryption_enabled    = optional(bool, true)<br>    parameter = optional(list(object({<br>      name  = string<br>      value = string<br>    })), [])<br>  })</pre> | <pre>{<br>  "allowed_security_group_ids": [],<br>  "apply_immediately": true,<br>  "at_rest_encryption_enabled": true,<br>  "automatic_failover_enabled": false,<br>  "availability_zones": null,<br>  "cluster_size": 3,<br>  "elasticache_subnet_group_name": null,<br>  "engine_version": "6.x",<br>  "family": "redis6.x",<br>  "instance_type": "cache.m5.large",<br>  "name": "fleet",<br>  "parameter": [],<br>  "replication_group_id": null,<br>  "subnets": null,<br>  "transit_encryption_enabled": true<br>}</pre> | no |
| <a name="input_vpc"></a> [vpc](#input\_vpc) | n/a | <pre>object({<br>    name                = optional(string, "fleet")<br>    cidr                = optional(string, "10.10.0.0/16")<br>    azs                 = optional(list(string), ["us-east-2a", "us-east-2b", "us-east-2c"])<br>    private_subnets     = optional(list(string), ["10.10.1.0/24", "10.10.2.0/24", "10.10.3.0/24"])<br>    public_subnets      = optional(list(string), ["10.10.11.0/24", "10.10.12.0/24", "10.10.13.0/24"])<br>    database_subnets    = optional(list(string), ["10.10.21.0/24", "10.10.22.0/24", "10.10.23.0/24"])<br>    elasticache_subnets = optional(list(string), ["10.10.31.0/24", "10.10.32.0/24", "10.10.33.0/24"])<br><br>    create_database_subnet_group          = optional(bool, false)<br>    create_database_subnet_route_table    = optional(bool, true)<br>    create_elasticache_subnet_group       = optional(bool, true)<br>    create_elasticache_subnet_route_table = optional(bool, true)<br>    enable_vpn_gateway                    = optional(bool, false)<br>    one_nat_gateway_per_az                = optional(bool, false)<br>    single_nat_gateway                    = optional(bool, true)<br>    enable_nat_gateway                    = optional(bool, true)<br>  })</pre> | <pre>{<br>  "azs": [<br>    "us-east-2a",<br>    "us-east-2b",<br>    "us-east-2c"<br>  ],<br>  "cidr": "10.10.0.0/16",<br>  "create_database_subnet_group": false,<br>  "create_database_subnet_route_table": true,<br>  "create_elasticache_subnet_group": true,<br>  "create_elasticache_subnet_route_table": true,<br>  "database_subnets": [<br>    "10.10.21.0/24",<br>    "10.10.22.0/24",<br>    "10.10.23.0/24"<br>  ],<br>  "elasticache_subnets": [<br>    "10.10.31.0/24",<br>    "10.10.32.0/24",<br>    "10.10.33.0/24"<br>  ],<br>  "enable_nat_gateway": true,<br>  "enable_vpn_gateway": false,<br>  "name": "fleet",<br>  "one_nat_gateway_per_az": false,<br>  "private_subnets": [<br>    "10.10.1.0/24",<br>    "10.10.2.0/24",<br>    "10.10.3.0/24"<br>  ],<br>  "public_subnets": [<br>    "10.10.11.0/24",<br>    "10.10.12.0/24",<br>    "10.10.13.0/24"<br>  ],<br>  "single_nat_gateway": true<br>}</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_byo-vpc"></a> [byo-vpc](#output\_byo-vpc) | n/a |
