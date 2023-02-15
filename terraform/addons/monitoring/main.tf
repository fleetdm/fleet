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
  alarm_actions       = [var.sns_topic_arn]
  ok_actions          = [var.sns_topic_arn]
  dimensions = {
    DBInstanceIdentifier = each.key
  }
}

resource "aws_db_event_subscription" "default" {
  count     = var.mysql_cluster_members == [] ? 0 : 1
  name      = "rds-event-sub-${var.customer_prefix}"
  sns_topic = var.sns_topic_arn

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
  alarm_actions       = [var.sns_topic_arn]
  ok_actions          = [var.sns_topic_arn]
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
  alarm_actions             = [var.sns_topic_arn]
  ok_actions                = [var.sns_topic_arn]
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
  alarm_actions       = [var.sns_topic_arn]
  ok_actions          = [var.sns_topic_arn]
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
  alarm_actions       = [var.sns_topic_arn]
  ok_actions          = [var.sns_topic_arn]

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
  alarm_actions       = [var.sns_topic_arn]
  ok_actions          = [var.sns_topic_arn]

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
  alarm_actions       = [var.sns_topic_arn]
  ok_actions          = [var.sns_topic_arn]

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
  alarm_actions             = [var.sns_topic_arn]
  ok_actions                = [var.sns_topic_arn]
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
  alarm_name                = "redis-replication-lag-${var.customer_prefix}"
  alarm_description         = "This metric is only applicable for a node running as a read replica. It represents how far behind, in seconds, the replica is in applying changes from the primary node. For Redis engine version 5.0.6 onwards, the lag can be measured in milliseconds."
  comparison_operator       = "GreaterThanUpperThreshold"
  evaluation_periods        = "3"
  threshold_metric_id       = "e1"
  alarm_actions             = [var.sns_topic_arn]
  ok_actions                = [var.sns_topic_arn]
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
  alarm_actions       = [var.sns_topic_arn]
  ok_actions          = [var.sns_topic_arn]

  dimensions = {
    CertificateArn = var.acm_certificate_arn
  }
}
