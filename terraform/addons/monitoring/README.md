# Monitoring addon
This addon enables Cloudwatch monitoring for Fleet.

This includes:

- 5XX Errors on ALB
- ECS Service Monitoring
- RDS Monitoring
- Redis Monitoring
- ACM Certificate Monitoring

# Preparation

Some of the for\_each and counts in this module cannot pre-determine the numbers until the `main` fleet module is applied.

You will need to `terraform apply -target module.main` prior applying monitoring assuming the use of a configuration matching the example at https://github.com/fleetdm/fleet/blob/main/terraform/example/main.tf.

# Example Configuration

This assumes your fleet module is `main` and is configured with it's default documentation.

See https://github.com/fleetdm/fleet/blob/main/terraform/example/main.tf for details.

```
module "monitoring" {
  source                      = "github.com/fleetdm/fleet//terraform/addons/monitoring?ref=main"
  fleet_ecs_service_name      = module.main.byo-vpc.byo-db.byo-ecs.service.name
  fleet_min_containers        = module.main.byo-vpc.byo-db.byo-ecs.service.desired_count
  alb_name                    = module.main.byo-vpc.byo-db.alb.lb_dns_name
  alb_target_group_name       = module.main.byo-vpc.byo-db.alb.target_group_names[0]
  alb_target_group_arn_suffix = module.main.byo-vpc.byo-db.alb.target_group_arn_suffixes[0]
  alb_arn_suffix              = module.main.byo-vpc.byo-db.alb.lb_arn_suffix
  default_sns_topic_arns      = [var.sns_topic_arn]
  mysql_cluster_members       = module.main.byo-vpc.rds.cluster_members
  redis_cluster_members       = module.main.byo-vpc.redis.member_clusters[0]
  acm_certificate_arn         = module.acm.acm_certificate_arn
}
```

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_cloudwatch_metric_alarm.acm_certificate_expired](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.alb_healthyhosts](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.cpu_utilization_too_high](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.lb](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.redis-current-connections](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.redis-database-memory-percentage](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.redis-replication-lag](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.redis_cpu](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.redis_cpu_engine_utilization](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_cloudwatch_metric_alarm.target_response_time](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_metric_alarm) | resource |
| [aws_db_event_subscription.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/db_event_subscription) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_acm_certificate_arn"></a> [acm\_certificate\_arn](#input\_acm\_certificate\_arn) | n/a | `string` | `null` | no |
| <a name="input_alb_arn_suffix"></a> [alb\_arn\_suffix](#input\_alb\_arn\_suffix) | n/a | `string` | `null` | no |
| <a name="input_alb_name"></a> [alb\_name](#input\_alb\_name) | n/a | `string` | `null` | no |
| <a name="input_alb_target_group_arn_suffix"></a> [alb\_target\_group\_arn\_suffix](#input\_alb\_target\_group\_arn\_suffix) | n/a | `string` | `null` | no |
| <a name="input_alb_target_group_name"></a> [alb\_target\_group\_name](#input\_alb\_target\_group\_name) | n/a | `string` | `null` | no |
| <a name="input_customer_prefix"></a> [customer\_prefix](#input\_customer\_prefix) | n/a | `string` | `"fleet"` | no |
| <a name="input_default_sns_topic_arns"></a> [default\_sns\_topic\_arns](#input\_default\_sns\_topic\_arns) | n/a | `list(string)` | `[]` | no |
| <a name="input_fleet_ecs_service_name"></a> [fleet\_ecs\_service\_name](#input\_fleet\_ecs\_service\_name) | n/a | `string` | `null` | no |
| <a name="input_fleet_min_containers"></a> [fleet\_min\_containers](#input\_fleet\_min\_containers) | n/a | `number` | `1` | no |
| <a name="input_mysql_cluster_members"></a> [mysql\_cluster\_members](#input\_mysql\_cluster\_members) | n/a | `list(string)` | `[]` | no |
| <a name="input_redis_cluster_members"></a> [redis\_cluster\_members](#input\_redis\_cluster\_members) | n/a | `list(string)` | `[]` | no |
| <a name="input_sns_topic_arns_map"></a> [sns\_topic\_arns\_map](#input\_sns\_topic\_arns\_map) | n/a | `map(list(string))` | `{}` | no |

## Outputs

No outputs.
