# External Vulnerability Scans addon
This addon creates an additional ECS service that only runs a single task, responsible for vuln processing. It receives
no web traffic. We utilize [current instance checks](https://fleetdm.com/docs/configuration/fleet-server-configuration#current-instance-checks) to make this happen. The advantages of this mechanism:

1. dedicating processing power to vuln processing
    2. ensures task responsible for vuln processing isn't also trying to serve web traffic
2. caching of vulnerability artifacts/dependencies

Usage is simplified by using the output from the fleet byo-ecs module (../terraform/byo-vpc/byo-db/byo-ecs/README.md)

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.11.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_ecs_service.fleet](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_service) | resource |
| [aws_ecs_task_definition.vuln-processing](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_task_definition) | resource |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_awslogs_config"></a> [awslogs\_config](#input\_awslogs\_config) | n/a | <pre>object({<br>    group  = string<br>    region = string<br>    prefix = string<br>  })</pre> | n/a | yes |
| <a name="input_customer_prefix"></a> [customer\_prefix](#input\_customer\_prefix) | n/a | `string` | `"fleet"` | no |
| <a name="input_ecs_cluster"></a> [ecs\_cluster](#input\_ecs\_cluster) | The ecs cluster module that is created by the byo-db module | `any` | n/a | yes |
| <a name="input_execution_iam_role_arn"></a> [execution\_iam\_role\_arn](#input\_execution\_iam\_role\_arn) | The ARN of the fleet execution role, this is necessary to pass role from ecs events | `any` | n/a | yes |
| <a name="input_fleet_config"></a> [fleet\_config](#input\_fleet\_config) | The root Fleet config object | `any` | n/a | yes |
| <a name="input_fleet_s3_software_installers_config"></a> [fleet\_s3\_software\_installers\_config](#input\_fleet\_s3\_software\_installers\_config) | use the output of the byo-vpc module with the same name | `map(string)` | n/a | yes |
| <a name="input_fleet_server_private_key_secret_arn"></a> [fleet\_server\_private\_key\_secret\_arn](#input\_fleet\_server\_private\_key\_secret\_arn) | The ARN of the secret that stores the Fleet private key | `string` | n/a | yes |
| <a name="input_security_groups"></a> [security\_groups](#input\_security\_groups) | n/a | `list(string)` | n/a | yes |
| <a name="input_subnets"></a> [subnets](#input\_subnets) | n/a | `list(string)` | n/a | yes |
| <a name="input_task_role_arn"></a> [task\_role\_arn](#input\_task\_role\_arn) | The ARN of the fleet task role, this is necessary to pass role from ecs events | `any` | n/a | yes |
| <a name="input_vuln_processing_cpu"></a> [vuln\_processing\_cpu](#input\_vuln\_processing\_cpu) | The amount of CPU to dedicate to the vuln processing command | `number` | `1024` | no |
| <a name="input_vuln_processing_memory"></a> [vuln\_processing\_memory](#input\_vuln\_processing\_memory) | The amount of memory to dedicate to the vuln processing command | `number` | `4096` | no |
| <a name="input_vuln_processing_task_cpu"></a> [vuln\_processing\_task\_cpu](#input\_vuln\_processing\_task\_cpu) | The amount of CPU to dedicate to the vuln processing task including sidecars | `number` | `1024` | no |
| <a name="input_vuln_processing_task_memory"></a> [vuln\_processing\_task\_memory](#input\_vuln\_processing\_task\_memory) | The amount of memory to dedicate to the vuln processing task including sidecars | `number` | `4096` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_extra_environment_variables"></a> [extra\_environment\_variables](#output\_extra\_environment\_variables) | n/a |
| <a name="output_vuln_service_arn"></a> [vuln\_service\_arn](#output\_vuln\_service\_arn) | n/a |
