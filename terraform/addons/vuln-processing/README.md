# vulnerability processing addon
This addon adds [external vulnerability processing](https://fleetdm.com/docs/using-fleet/vulnerability-processing#advanced-configuration) to the Fleet deployment.

Be sure to set `FLEET_VULNERABILITIES_DISABLE_SCHEDULE = "true"` or use this modules' `fleet_extra_environment_variables` output to configure
your Fleet server deployment.

Below is an example implementation of the module:

```
module "vulnerability_processing" {
  source                     = "github.com/fleetdm/fleet//terraform/addons/vuln-processing?ref=main"
  customer_prefix = "fleet"
  ecs_cluster     = module.main.byo-vpc.byo-db.byo-ecs.cluster.cluster_arn
  vpc_id          = module.main.vpc.vpc_id
  fleet_config = {
    image = "fleetdm/fleet:v4.28.1"
    database = {
      password_secret_arn = module.main.byo-vpc.secrets.secret_arns["${var.rds_config.name}-database-password"]
      user                = module.main.byo-vpc.rds.db_instance_username
      address             = "${module.main.byo-vpc.rds.db_instance_endpoint}:${module.main.byo-vpc.rds.db_instance_port}"
      database            = module.main.byo-vpc.rds.db_instance_name
    }
    extra_environment_variables = {
      FLEET_LOGGING_DEBUG = "true"
      FLEET_LOGGING_JSON  = "true"
    }
    extra_secrets = {
      // FLEET_LICENSE_KEY: "secret_manager_license_key_arn" // note needed for some feature of vuln processing
    }
    networking = {
      subnets         = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
      security_groups = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
    }
  }
}
```

## Requirements

[VPC DNS Hostnames](https://docs.aws.amazon.com/vpc/latest/userguide/vpc-dns.html#vpc-dns-hostnames) must be enabled for proper communication to EFS mounted volumes.

## Providers

| Name                                              | Version |
|---------------------------------------------------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | n/a     |

## Modules

No modules.

## Resources

| Name                                                                                                                                               | Type     |
|----------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| [aws_ecs_task_definition.vuln-data-stream](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_task_definition)        | resource |
| [aws_ecs_task_definition.vuln-processing](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_task_definition)         | resource |
| [aws_efs_file_system.vuln](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/efs_file_system)                            | resource |
| [aws_efs_mount_target.vuln](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/efs_mount_target)                          | resource |
| [aws_cloudwatch_event_rule.vuln_processing](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_event_rule)     | resource |
| [aws_cloudwatch_event_target.vuln_processing](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_event_target) | resource |
| [aws_security_group.efs_security_group](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group)                | resource |
| [aws_iam_role.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)                                          | resource |
| [aws_iam_role_policy_attachment.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy_attachment)           | resource |
| [aws_iam_role.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)                                          | resource |


## Inputs

| Name                                                                              | Description                                                                                                                           | Type     | Default   | Required |
|-----------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------|----------|-----------|:--------:|
| <a name="input_customer_prefix"></a> [customer\_prefix](#input\_customer\_prefix) | customer prefix to use to namespace all resources                                                                                     | `string` | `"fleet"` |    no    |
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster)             | ECS cluster ARN                                                                                                                       | `string` | n/a       |   yes    |
| <a name="input_vpc_id"></a> [vpc\_id](#input\_vpc\_id)                            | n/a                                                                                                                                   | `string` | n/a       |   yes    |
| <a name="input_fleet_config"></a> [fleet\_config](#input\_fleet\_config)          | The configuration object for Fleet itself. Fields that default to null will have their respective resources created if not specified. | `object` | no        |   yes    |

## Outputs

No outputs.
