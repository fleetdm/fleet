resource "aws_wafv2_rule_group" "main" {
  name     = var.name
  scope    = "REGIONAL"
  capacity = 2

  rule {
    name     = "countries"
    priority = 1

    action {
      block {}
    }

    statement {
      geo_match_statement {
        country_codes = var.blocked_countries
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = false
      metric_name                = var.name
      sampled_requests_enabled   = false
    }
  }

  rule {
    name     = "specific"
    priority = 2

    action {
      block {}
    }

    statement {
      ip_set_reference_statement {
        arn = aws_wafv2_ip_set.main.arn
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = false
      metric_name                = var.name
      sampled_requests_enabled   = false
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = false
    metric_name                = var.name
    sampled_requests_enabled   = false
  }
}

resource "aws_wafv2_ip_set" "main" {
  name               = var.name
  scope              = "REGIONAL"
  ip_address_version = "IPV4"
  addresses          = var.blocked_addresses
}

resource "aws_wafv2_web_acl" "main" {
  name  = var.name
  scope = "REGIONAL"

  default_action {
    allow {}
  }

  rule {
    name     = "rule-1"
    priority = 1

    override_action {
      none {}
    }

    statement {
      rule_group_reference_statement {
        arn = aws_wafv2_rule_group.main.arn
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = false
      metric_name                = var.name
      sampled_requests_enabled   = false
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = false
    metric_name                = var.name
    sampled_requests_enabled   = false
  }
}

resource "aws_wafv2_web_acl_association" "main" {
  resource_arn = var.lb_arn
  web_acl_arn  = aws_wafv2_web_acl.main.arn
}
