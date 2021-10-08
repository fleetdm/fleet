// sns topic to send cloudwatch alarms to
resource "aws_sns_topic" "cloudwatch_alarm_topic" {
  name_prefix = "fleet-cloudwatch-alarm"
}

resource "aws_sns_topic_policy" "default" {
  arn    = aws_sns_topic.cloudwatch_alarm_topic.arn
  policy = data.aws_iam_policy_document.sns_topic_policy.json
}

data "aws_iam_policy_document" "sns_topic_policy" {
  statement {
    sid = "AllowManageSNS"

    actions = [
      "sns:Subscribe",
      "sns:SetTopicAttributes",
      "sns:RemovePermission",
      "sns:Receive",
      "sns:Publish",
      "sns:ListSubscriptionsByTopic",
      "sns:GetTopicAttributes",
      "sns:DeleteTopic",
      "sns:AddPermission",
    ]

    effect    = "Allow"
    resources = [aws_sns_topic.cloudwatch_alarm_topic.arn]

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceOwner"

      values = [data.aws_caller_identity.current.account_id]

    }
  }

  statement {
    sid       = "Allow CloudwatchEvents"
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.cloudwatch_alarm_topic.arn]

    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
  }

  statement {
    sid       = "Allow RDS Event Notification"
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.cloudwatch_alarm_topic.arn]

    principals {
      type        = "Service"
      identifiers = ["rds.amazonaws.com"]
    }
  }
}


// Database alarms
resource "aws_cloudwatch_metric_alarm" "cpu_utilization_too_high" {
  for_each            = toset(module.aurora_mysql.rds_cluster_instance_ids)
  alarm_name          = "cpu_utilization_too_high-${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/RDS"
  period              = "300"
  statistic           = "Average"
  threshold           = 80
  alarm_description   = "Average database CPU utilization over last 5 minutes too high"
  alarm_actions       = [aws_sns_topic.cloudwatch_alarm_topic.arn]
  ok_actions          = [aws_sns_topic.cloudwatch_alarm_topic.arn]
  dimensions = {
    DBInstanceIdentifier = each.key
  }
}

resource "aws_db_event_subscription" "default" {
  name      = "rds-event-sub"
  sns_topic = aws_sns_topic.cloudwatch_alarm_topic.arn

  source_type = "db-instance"
  source_ids  = module.aurora_mysql.rds_cluster_instance_ids

  event_categories = [
    "failover",
    "failure",
    "low storage",
    "maintenance",
    "notification",
    "recovery",
  ]

  depends_on = [
    aws_sns_topic_policy.default
  ]
}