# Monitoring addon
This addon enables Cloudwatch monitoring for Fleet.

This includes:

- 5XX Errors on ALB
- ECS Service Monitoring
- RDS Monitoring
- Redis Monitoring
- ACM Certificate Monitoring
- A custom Lambda to check the Fleet DB for Cron runs

# Preparation

Some of the for\_each and counts in this module cannot pre-determine the numbers until the `main` fleet module is applied.

You will need to `terraform apply -target module.main` prior applying monitoring assuming the use of a configuration matching the example at https://github.com/fleetdm/fleet/blob/main/terraform/example/main.tf.

Multiple alb support was added in order to allow monitoring `saml-auth-proxy`.  See https://github.com/fleetdm/fleet/tree/main/terraform/addons/saml-auth-proxy

# Example configuration

This assumes your fleet module is `main` and is configured with it's default documentation.

 https://github.com/fleetdm/fleet/blob/main/terraform/example/main.tf for details.

Note if you haven't specified defined `local.customer` or customized service names, the default is "fleet" for anywhere that `local.customer` is specified below.

```
module "monitoring" {
  source                 = "github.com/fleetdm/fleet//terraform/addons/monitoring?ref=tf-mod-addon-monitoring-v1.4.0"
  customer_prefix        = local.customer
  fleet_ecs_service_name = module.main.byo-vpc.byo-db.byo-ecs.service.name
  albs = [
    {
      name                    = module.main.byo-vpc.byo-db.alb.lb_dns_name,
      target_group_name       = module.main.byo-vpc.byo-db.alb.target_group_names[0]
      target_group_arn_suffix = module.main.byo-vpc.byo-db.alb.target_group_arn_suffixes[0]
      arn_suffix              = module.main.byo-vpc.byo-db.alb.lb_arn_suffix
      ecs_service_name        = module.main.byo-vpc.byo-db.byo-ecs.service.name
      min_containers          = module.main.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
      alert_thresholds = {
        HTTPCode_ELB_5XX_Count = {
          period    = 3600
          threshold = 2
        },
        HTTPCode_Target_5XX_Count = {
          period    = 120
          threshold = 0
        }
      }
    },
  ]
  sns_topic_arns_map = {
    alb_httpcode_5xx = [var.sns_topic_arn]
    cron_monitoring  = [var.sns_topic_arn]
    cron_job_failure_monitoring  = [var.sns_another_topic_arn]
  }
  mysql_cluster_members = module.main.byo-vpc.rds.cluster_members
  # The cloudposse module seems to have a nested list here.
  redis_cluster_members = module.main.byo-vpc.redis.member_clusters[0]
  acm_certificate_arn   = module.acm.acm_certificate_arn
  cron_monitoring = {
    mysql_host                 = module.main.byo-vpc.rds.cluster_reader_endpoint
    mysql_database             = module.main.byo-vpc.rds.cluster_database_name
    mysql_user                 = module.main.byo-vpc.rds.cluster_master_username
    mysql_password_secret_name = module.main.byo-vpc.secrets.secret_ids["${local.customer}-database-password"]
    rds_security_group_id      = module.main.byo-vpc.rds.security_group_id
    subnet_ids                 = module.main.vpc.private_subnets
    vpc_id                     = module.main.vpc.vpc_id
    # Format of https://pkg.go.dev/time#ParseDuration
    delay_tolerance = "4h"
    # Interval format for: https://docs.aws.amazon.com/scheduler/latest/UserGuide/schedule-types.html#rate-based
    run_interval          = "1 hour"
    log_retention_in_days = 365
  }
}
```

# SNS topic ARNs map

Valid targets for `sns_topic_arns_map`:

 - acm\_certificate\_expired
 - alb\_helthyhosts
 - alb\_httpcode\_5xx
 - backend\_response\_time
 - cron\_monitoring (notifications about failures in the cron scheduler)
 - cron\_job\_failure\_monitoring (notifications about errors in individual cron jobs - defaults to value of `cron_monitoring`)
 - rds\_cpu\_untilizaton\_too\_high
 - rds\_db\_event\_subscription
 - redis\_cpu\_engine\_utilization
 - redis\_cpu\_utilization
 - redis\_current\_connections
 - redis\_database\_memory\_percentage
 - redis\_replication\_lag

If you want to publish to all, use `default_sns_topic_arns` instead and include your notification ARNs there.

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_archive"></a> [archive](#provider\_archive) | 2.7.0 |
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.81.0 |
| <a name="provider_null"></a> [null](#provider\_null) | 3.2.3 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_cloudwatch_event_rule.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_event_rule) | resource |
| [aws_cloudwatch_event_target.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_event_target) | resource |
| [aws_cloudwatch_log_group.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_log_group) | resource |
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
| [aws_iam_policy.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.cron_monitoring_lambda_managed](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_lambda_function.cron_monitoring](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lambda_function) | resource |
| [aws_lambda_permission.cron_monitoring_cloudwatch](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lambda_permission) | resource |
| [aws_security_group.cron_monitoring](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |
| [aws_security_group_rule.cron_monitoring_to_rds](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group_rule) | resource |
| [null_resource.cron_monitoring_build](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [archive_file.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/archive/latest/docs/data-sources/file) | data source |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_iam_policy_document.cron_monitoring_lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.cron_monitoring_lambda_assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |
| [aws_secretsmanager_secret.mysql_database_password](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/secretsmanager_secret) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_acm_certificate_arn"></a> [acm\_certificate\_arn](#input\_acm\_certificate\_arn) | n/a | `string` | `null` | no |
| <a name="input_albs"></a> [albs](#input\_albs) | n/a | <pre>list(object({<br/>    name                    = string<br/>    arn_suffix              = string<br/>    target_group_name       = string<br/>    target_group_arn_suffix = string<br/>    min_containers          = optional(string, 1)<br/>    ecs_service_name        = string<br/>    alert_thresholds = optional(<br/>      object({<br/>        HTTPCode_ELB_5XX_Count = object({<br/>          period    = number<br/>          threshold = number<br/>        })<br/>        HTTPCode_Target_5XX_Count = object({<br/>          period    = number<br/>          threshold = number<br/>        })<br/>      }),<br/>      {<br/>        HTTPCode_ELB_5XX_Count = {<br/>          period    = 120<br/>          threshold = 0<br/>        },<br/>        HTTPCode_Target_5XX_Count = {<br/>          period    = 120<br/>          threshold = 0<br/>        }<br/>      }<br/>    )<br/>  }))</pre> | `[]` | no |
| <a name="input_cron_monitoring"></a> [cron\_monitoring](#input\_cron\_monitoring) | n/a | <pre>object({<br/>    mysql_host                 = string<br/>    mysql_database             = string<br/>    mysql_user                 = string<br/>    mysql_password_secret_name = string<br/>    vpc_id                     = string<br/>    subnet_ids                 = list(string)<br/>    rds_security_group_id      = string<br/>    delay_tolerance            = string<br/>    run_interval               = string<br/>    log_retention_in_days      = optional(number, 7)<br/>  })</pre> | `null` | no |
| <a name="input_customer_prefix"></a> [customer\_prefix](#input\_customer\_prefix) | n/a | `string` | `"fleet"` | no |
| <a name="input_default_sns_topic_arns"></a> [default\_sns\_topic\_arns](#input\_default\_sns\_topic\_arns) | n/a | `list(string)` | `[]` | no |
| <a name="input_fleet_ecs_service_name"></a> [fleet\_ecs\_service\_name](#input\_fleet\_ecs\_service\_name) | n/a | `string` | `null` | no |
| <a name="input_mysql_cluster_members"></a> [mysql\_cluster\_members](#input\_mysql\_cluster\_members) | n/a | `list(string)` | `[]` | no |
| <a name="input_redis_cluster_members"></a> [redis\_cluster\_members](#input\_redis\_cluster\_members) | n/a | `list(string)` | `[]` | no |
| <a name="input_sns_topic_arns_map"></a> [sns\_topic\_arns\_map](#input\_sns\_topic\_arns\_map) | n/a | `map(list(string))` | `{}` | no |

## Outputs

No outputs.
