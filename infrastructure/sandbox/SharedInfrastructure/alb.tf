resource "aws_lb" "main" {
  name                       = var.prefix
  internal                   = false
  load_balancer_type         = "application"
  security_groups            = [aws_security_group.lb.id]
  subnets                    = var.vpc.public_subnets
  enable_deletion_protection = true

  access_logs {
    bucket  = module.s3_bucket_for_logs.s3_bucket_id
    prefix  = var.prefix
    enabled = true
  }
}

output "lb" {
  value = aws_lb.main
}

resource "aws_security_group" "lb" {
  name        = "${var.prefix}-lb"
  vpc_id      = var.vpc.vpc_id
  description = "${var.prefix}-lb"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

resource "aws_lb_listener" "main" {
  load_balancer_arn = aws_lb.main.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS-1-2-Ext-2018-06"
  certificate_arn   = aws_acm_certificate.main.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.eks.arn
  }
}

resource "aws_lb_listener" "redirect" {
  load_balancer_arn = aws_lb.main.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "redirect"
    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

output "alb_listener" {
  value = aws_lb_listener.main
}

resource "aws_acm_certificate" "main" {
  domain_name               = "*.${var.base_domain}"
  subject_alternative_names = [var.base_domain]
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate_validation" "main" {
  certificate_arn         = aws_acm_certificate.main.arn
  validation_record_fqdns = [for r in cloudflare_record.cert : r.hostname]
}

data "cloudflare_zone" "main" {
  name = "fleetdm.com"
}

resource "cloudflare_record" "cert" {
  for_each = { for o in aws_acm_certificate.main.domain_validation_options.* : o.resource_record_name => o... }
  zone_id  = data.cloudflare_zone.main.id
  name     = replace(each.value[0].resource_record_name, ".fleetdm.com.", "")
  type     = each.value[0].resource_record_type
  value    = replace(each.value[0].resource_record_value, "/.$/", "")
  ttl      = 1
  proxied  = false
}

resource "cloudflare_record" "main" {
  zone_id = data.cloudflare_zone.main.id
  name    = local.env_specific[data.aws_caller_identity.current.account_id]["dns_name"]
  type    = "CNAME"
  value   = aws_lb.main.dns_name
  proxied = false
}

resource "cloudflare_record" "wildcard" {
  zone_id = data.cloudflare_zone.main.id
  name    = "*.${local.env_specific[data.aws_caller_identity.current.account_id]["dns_name"]}"
  type    = "CNAME"
  value   = aws_lb.main.dns_name
  proxied = false
}

module "s3_bucket_for_logs" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "3.6.0"

  bucket = "${var.prefix}-alb-logs"
  acl    = "log-delivery-write"

  # Allow deletion of non-empty bucket
  force_destroy = true

  attach_elb_log_delivery_policy        = true # Required for ALB logs
  attach_lb_log_delivery_policy         = true # Required for ALB/NLB logs
  attach_deny_insecure_transport_policy = true
  attach_require_latest_tls_policy      = true
  block_public_acls                     = true
  block_public_policy                   = true
  ignore_public_acls                    = true
  restrict_public_buckets               = true
  server_side_encryption_configuration = {
    rule = {
      apply_server_side_encryption_by_default = {
        kms_master_key_id = var.kms_key.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
  lifecycle_rule = [
    {
      id      = "log"
      enabled = true

      transition = [
        {
          days          = 30
          storage_class = "ONEZONE_IA"
        }
      ]
      expiration = {
        days                         = 90
        expired_object_delete_marker = true
      }
      noncurrent_version_expiration = {
        newer_noncurrent_versions = 5
        days                      = 30
      }
    }
  ]
}

output "access_logs_s3_bucket" {
  value = module.s3_bucket_for_logs
}

resource "aws_athena_database" "logs" {
  name   = replace("${var.prefix}-alb-logs", "-", "_")
  bucket = module.athena-s3-bucket.s3_bucket_id
}

module "athena-s3-bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "3.6.0"

  bucket = "${var.prefix}-alb-logs-athena"
  acl    = "log-delivery-write"

  # Allow deletion of non-empty bucket
  force_destroy = true

  attach_elb_log_delivery_policy        = true # Required for ALB logs
  attach_lb_log_delivery_policy         = true # Required for ALB/NLB logs
  attach_deny_insecure_transport_policy = true
  attach_require_latest_tls_policy      = true
  block_public_acls                     = true
  block_public_policy                   = true
  ignore_public_acls                    = true
  restrict_public_buckets               = true
  server_side_encryption_configuration = {
    rule = {
      apply_server_side_encryption_by_default = {
        kms_master_key_id = var.kms_key.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
  lifecycle_rule = [
    {
      id      = "log"
      enabled = true

      transition = [
        {
          days          = 30
          storage_class = "ONEZONE_IA"
        }
      ]
      expiration = {
        days                         = 90
        expired_object_delete_marker = true
      }
      noncurrent_version_expiration = {
        newer_noncurrent_versions = 5
        days                      = 30
      }
    }
  ]
}

resource "aws_athena_workgroup" "logs" {
  name = "${var.prefix}-logs"

  configuration {
    enforce_workgroup_configuration    = true
    publish_cloudwatch_metrics_enabled = true

    result_configuration {
      output_location = "s3://${module.athena-s3-bucket.s3_bucket_id}/output/"

      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = var.kms_key.arn
      }
    }
  }
}
