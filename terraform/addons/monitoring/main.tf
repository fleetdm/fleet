// Database alarms
resource "aws_cloudwatch_metric_alarm" "cpu_utilization_too_high" {
  for_each            = toset(var.mysql_cluster_members)
  alarm_name          = "rds_cpu_utilization_too_high-${var.customer_prefix}-${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/RDS"
  period              = "300"
  statistic           = "Average"
  threshold           = 80
  alarm_description   = "Average database CPU utilization over last 5 minutes too high"
  alarm_actions       = lookup(var.sns_topic_arns_map, "rds_cpu_untilizaton_too_high", var.default_sns_topic_arns)
  ok_actions          = lookup(var.sns_topic_arns_map, "rds_cpu_untilizaton_too_high", var.default_sns_topic_arns)
  dimensions = {
    DBInstanceIdentifier = each.key
  }
}

resource "aws_db_event_subscription" "default" {
  count     = length(var.mysql_cluster_members) == 0 || (contains(keys(var.sns_topic_arns_map), "rds_db_event_subscription") == false && length(var.default_sns_topic_arns) == 0) ? 0 : 1
  name      = "rds-event-sub-${var.customer_prefix}"
  sns_topic = try(var.sns_topic_arns_map.rds_db_event_subscription[0], var.default_sns_topic_arns[0])

  source_type = "db-instance"
  source_ids  = var.mysql_cluster_members

  event_categories = [
    "failover",
    "failure",
    "low storage",
    "maintenance",
    "notification",
    "recovery",
  ]

}

// ECS Alarms
resource "aws_cloudwatch_metric_alarm" "alb_healthyhosts" {
  count               = var.alb_target_group_arn_suffix == null || var.alb_arn_suffix == null ? 0 : 1
  alarm_name          = "backend-healthyhosts-${var.customer_prefix}"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "HealthyHostCount"
  namespace           = "AWS/ApplicationELB"
  period              = "60"
  statistic           = "Minimum"
  threshold           = var.fleet_min_containers
  alarm_description   = "This alarm indicates the number of Healthy Fleet hosts is lower than expected. Please investigate the load balancer \"${var.alb_name}\" or the target group \"${var.alb_target_group_name}\" and the fleet backend service \"${var.fleet_ecs_service_name}\""
  actions_enabled     = "true"
  alarm_actions       = lookup(var.sns_topic_arns_map, "alb_helthyhosts", var.default_sns_topic_arns)
  ok_actions          = lookup(var.sns_topic_arns_map, "alb_helthyhosts", var.default_sns_topic_arns)
  dimensions = {
    TargetGroup  = var.alb_target_group_arn_suffix
    LoadBalancer = var.alb_arn_suffix
  }
}

// alarm for target response time (anomaly detection)
resource "aws_cloudwatch_metric_alarm" "target_response_time" {
  count                     = var.alb_target_group_arn_suffix == null || var.alb_arn_suffix == null ? 0 : 1
  alarm_name                = "backend-target-response-time-${var.customer_prefix}"
  comparison_operator       = "GreaterThanUpperThreshold"
  evaluation_periods        = "2"
  threshold_metric_id       = "e1"
  alarm_description         = "This alarm indicates the Fleet server response time is greater than it usually is. Please investigate the ecs service \"${var.fleet_ecs_service_name}\" because the backend might need to be scaled up."
  alarm_actions             = lookup(var.sns_topic_arns_map, "backend_response_time", var.default_sns_topic_arns)
  ok_actions                = lookup(var.sns_topic_arns_map, "backend_response_time", var.default_sns_topic_arns)
  insufficient_data_actions = []

  metric_query {
    id          = "e1"
    expression  = "ANOMALY_DETECTION_BAND(m1)"
    label       = "TargetResponseTime (Expected)"
    return_data = "true"
  }

  metric_query {
    id          = "m1"
    return_data = "true"
    metric {
      metric_name = "TargetResponseTime"
      namespace   = "AWS/ApplicationELB"
      period      = "120"
      stat        = "p99"
      unit        = "Count"

      dimensions = {
        TargetGroup  = var.alb_target_group_arn_suffix
        LoadBalancer = var.alb_arn_suffix
      }
    }
  }
}

resource "aws_cloudwatch_metric_alarm" "lb" {
  for_each            = var.alb_target_group_arn_suffix == null ? toset([]) : toset(["HTTPCode_ELB_5XX_Count", "HTTPCode_Target_5XX_Count"])
  alarm_name          = "${var.customer_prefix}-lb-${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = each.key
  namespace           = "AWS/ApplicationELB"
  period              = "120"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "This alarm indicates there are an abnormal amount of 5XX responses.  Either the lb cannot talk with the Fleet backend target or Fleet is returning an error."
  alarm_actions       = lookup(var.sns_topic_arns_map, "alb_httpcode_5xx", var.default_sns_topic_arns)
  ok_actions          = lookup(var.sns_topic_arns_map, "alb_httpcode_5xx", var.default_sns_topic_arns)
  treat_missing_data  = "notBreaching"
  dimensions = {
    LoadBalancer = var.alb_arn_suffix
  }
}


// Elasticache (redis) alerts https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.WhichShouldIMonitor.html
resource "aws_cloudwatch_metric_alarm" "redis_cpu" {
  for_each            = toset(var.redis_cluster_members)
  alarm_name          = "redis-cpu-utilization-${each.key}-${var.customer_prefix}"
  alarm_description   = "Redis cluster CPU utilization node ${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ElastiCache"
  period              = "300"
  statistic           = "Average"
  alarm_actions       = lookup(var.sns_topic_arns_map, "redis_cpu_utilization", var.default_sns_topic_arns)
  ok_actions          = lookup(var.sns_topic_arns_map, "redis_cpu_utilization", var.default_sns_topic_arns)

  threshold = "70"

  dimensions = {
    CacheClusterId = each.key
  }

}

resource "aws_cloudwatch_metric_alarm" "redis_cpu_engine_utilization" {
  for_each            = toset(var.redis_cluster_members)
  alarm_name          = "redis-cpu-engine-utilization-${each.key}-${var.customer_prefix}"
  alarm_description   = "Redis cluster CPU Engine utilization node ${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "EngineCPUUtilization"
  namespace           = "AWS/ElastiCache"
  period              = "300"
  statistic           = "Average"
  alarm_actions       = lookup(var.sns_topic_arns_map, "redis_cpu_engine_utilization", var.default_sns_topic_arns)
  ok_actions          = lookup(var.sns_topic_arns_map, "redis_cpu_engine_utilization", var.default_sns_topic_arns)

  threshold = "25"

  dimensions = {
    CacheClusterId = each.key
  }

}

resource "aws_cloudwatch_metric_alarm" "redis-database-memory-percentage" {
  for_each            = toset(var.redis_cluster_members)
  alarm_name          = "redis-database-memory-percentage-${each.key}-${var.customer_prefix}"
  alarm_description   = "Percentage of the memory available for the cluster that is in use. This is calculated using used_memory/maxmemory."
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "DatabaseMemoryUsagePercentage"
  namespace           = "AWS/ElastiCache"
  period              = "300"
  statistic           = "Average"
  alarm_actions       = lookup(var.sns_topic_arns_map, "redis_database_memory_percentage", var.default_sns_topic_arns)
  ok_actions          = lookup(var.sns_topic_arns_map, "redis_database_memory_percentage", var.default_sns_topic_arns)

  threshold = "80"

  dimensions = {
    CacheClusterId = each.key
  }

}

resource "aws_cloudwatch_metric_alarm" "redis-current-connections" {
  for_each                  = toset(var.redis_cluster_members)
  alarm_name                = "redis-current-connections-${each.key}-${var.customer_prefix}"
  alarm_description         = "Redis current connections for node ${each.key}"
  comparison_operator       = "LessThanLowerOrGreaterThanUpperThreshold"
  evaluation_periods        = "5"
  threshold_metric_id       = "e1"
  alarm_actions             = lookup(var.sns_topic_arns_map, "redis_current_connections", var.default_sns_topic_arns)
  ok_actions                = lookup(var.sns_topic_arns_map, "redis_current_connections", var.default_sns_topic_arns)
  insufficient_data_actions = []

  metric_query {
    id          = "e1"
    expression  = "ANOMALY_DETECTION_BAND(m1,20)"
    label       = "Current Connections (Expected)"
    return_data = "true"
  }

  metric_query {
    id          = "m1"
    return_data = "true"
    metric {
      metric_name = "CurrConnections"
      namespace   = "AWS/ElastiCache"
      period      = "600"
      stat        = "Average"
      unit        = "Count"

      dimensions = {
        CacheClusterId = each.key
      }
    }
  }
}

resource "aws_cloudwatch_metric_alarm" "redis-replication-lag" {
  for_each                  = toset(var.redis_cluster_members)
  alarm_name                = "redis-replication-lag-${each.key}-${var.customer_prefix}"
  alarm_description         = "This metric is only applicable for a node running as a read replica. It represents how far behind, in seconds, the replica is in applying changes from the primary node. For Redis engine version 5.0.6 onwards, the lag can be measured in milliseconds."
  comparison_operator       = "GreaterThanUpperThreshold"
  evaluation_periods        = "3"
  threshold_metric_id       = "e1"
  alarm_actions             = lookup(var.sns_topic_arns_map, "redis_replication_lag", var.default_sns_topic_arns)
  ok_actions                = lookup(var.sns_topic_arns_map, "redis_replication_lag", var.default_sns_topic_arns)
  insufficient_data_actions = []

  metric_query {
    id          = "e1"
    expression  = "ANOMALY_DETECTION_BAND(m1)"
    label       = "ReplicationLag (expected)"
    return_data = "true"
  }

  metric_query {
    id          = "m1"
    return_data = "true"
    metric {
      metric_name = "ReplicationLag"
      namespace   = "AWS/ElastiCache"
      period      = "300"
      stat        = "p90"

      dimensions = {
        CacheClusterId = each.key
      }
    }
  }
}

// ACM Certificate Manager
resource "aws_cloudwatch_metric_alarm" "acm_certificate_expired" {
  count               = var.acm_certificate_arn == null ? 0 : 1
  alarm_name          = "acm-cert-expiry-${var.customer_prefix}"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = "1"
  period              = "86400" // 1 day in seconds
  threshold           = 30      // days
  statistic           = "Average"
  namespace           = "AWS/CertificateManager"
  metric_name         = "DaysToExpiry"
  actions_enabled     = "true"
  alarm_description   = "ACM Certificate will expire soon"
  alarm_actions       = lookup(var.sns_topic_arns_map, "acm_certificate_expired", var.default_sns_topic_arns)
  ok_actions          = lookup(var.sns_topic_arns_map, "acm_certificate_expired", var.default_sns_topic_arns)

  dimensions = {
    CertificateArn = var.acm_certificate_arn
  }
}

// Cron Monitoring
locals {
  cron_monitoring_filename = "${path.module}/lambda.tar.gz"

}

resource "null_resource" "cron_monitoring_build" {
  provisioner "local-exec" {
    working_dir = "${path.module}/lambda"
    command     = <<-EOT
      go get
      go build .
    EOT
  }
}

data "archive_file" "cron_monitoring_lambda" {
  depends_on   = [null_resource.cron_monitoring_sync_build]
  type         = "zip"
  output_path  = "${path.module}/lambda/.lambda.zip"
  source_file  = "${path.module}/lambda/lambda"
}

resource "aws_lambda_function" "cron_monitoring" {

  depends_on = [
    null_resource.cron_monitoring_build,
    data.archive_file.cron_monitoring_lambda
  ]

  function_name                  = "${var.customer_prefix}_cron_monitoring"
  runtime                        = "go1.x"
  memory_size                    = 256
  timeout                        = 300
  package_type                   = "Zip"
  filename                       = data.archive_file.cron_monitoring_lambda.output_path
  handler                        = "/lambda"
  reserved_concurrent_executions = 1
  description                    = "This function has the ability to log into a production database and validate that the Fleet crons are running properly"
  tracing_config {
    mode = "Active"
  }

  role = aws_iam_role.cron_monitoring_lambda.arn

  environment {
    variables = {
      MYSQL_HOST                 = var.cron_monitoring.mysql_host
      MYSQL_DATABASE             = var.cron_monitoring.mysql_database
      MYSQL_USER                 = var.cron_monitoring.mysql_user
      MYSQL_PASSWORD_SECRET_NAME = var.cron_monitoring.mysql_password_secret.name
      SNS_TOPIC_ARN              = var.cron_monitoring.sns_topic_arn
      FLEET_ENV                  = var.customer_prefix
    }
  }

}

// Lambda IAM
data "aws_iam_policy_document" "cron_monitoring_lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "cron_monitoring_lambda" {
  role       = aws_iam_role.cron_monitoring_lambda.id
  policy_arn = aws_iam_policy.cron_monitoring_lambda.arn
}

resource "aws_iam_role_policy_attachment" "cron_monitoring_lambda_managed" {
  for_each   = toset(local.idp_scim_sync_iam_managed_policies)
  role       = aws_iam_role.lambda.id
  policy_arn = each.key
}

resource "aws_iam_policy" "cron_monitoring_lambda" {
  name     = "${var.customer_prefix}-cron-monitoring"
  policy   = data.aws_iam_policy_document.cron_monitoring_lambda.json
}

resource "aws_iam_role" "cron_monitoring_lambda" {
  name               = "idp-scim-sync-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
}

data "aws_iam_policy_document" "cron_monitoring_lambda" {
  statement {

    sid = "SSMGetParameterPolicy"

    actions = [
      "secretsmanager:GetResourcePolicy",
      "secretsmanager:GetSecretValue"
    ]

    resources = [var.cron_monitoring.mysql_password_secret.arn]

    effect = "Allow"

  }

  statement {
    sid = "SNSPublish"

    actions = [
      "sns:Publish"
    ]

    resources = [var.sns_topic_arn]

    effect = "Allow"
  }

}

resource "aws_cloudwatch_log_group" "cron_monitoring_lambda" {
  name              = "/aws/lambda/${var.customer_prefix}-cron-monitoring"
  retention_in_days = 7

}

resource "aws_cloudwatch_event_rule" "cron_monitoring_lambda" {
  name                = "${var.customer_prefix}-cron-monitoring"
  schedule_expression = "rate(2 hours)"
  is_enabled          = true
}

resource "aws_cloudwatch_event_target" "cron_monitoring_lambda" {
  rule     = aws_cloudwatch_event_rule.cron_monitoring_lambda.name
  arn      = aws_lambda_function.cron_monitoring.arn
}

resource "aws_lambda_permission" "cron_monitoring_cloudwatch" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cron_monitoring.id
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.cron_monitoring_lambda.arn
}
