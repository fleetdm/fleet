module "s3_bucket_for_logs" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "3.15.1"

  bucket = "fleet-loadtesting-alb-logs"

  # Allow deletion of non-empty bucket
  force_destroy = true

  attach_elb_log_delivery_policy        = true # Required for ALB logs
  attach_lb_log_delivery_policy         = true # Required for ALB/NLB logs
  attach_deny_insecure_transport_policy = true
  attach_require_latest_tls_policy      = true
  # attach_policy                         = var.extra_s3_log_policies != []
  # policy                                = var.extra_s3_log_policies != [] ? data.aws_iam_policy_document.s3_log_bucket[0].json : null
  block_public_acls                     = true
  block_public_policy                   = true
  ignore_public_acls                    = true
  restrict_public_buckets               = true
  acl                                   = "private"
  control_object_ownership              = true
  object_ownership                      = "ObjectWriter"

  server_side_encryption_configuration = {
    rule = {
      bucket_key_enabled = true
      apply_server_side_encryption_by_default = {
        sse_algorithm = "AES256"
      }
    }
  }
  lifecycle_rule = [
    {
      id      = "log"
      enabled = true

      transition = [
        {
          days          = 90
          storage_class = "ONEZONE_IA"
        }
      ]
      expiration = {
        days = 365
        # Always resets to false anyhow showing terraform changes constantly
        expired_object_delete_marker = false
      }
      noncurrent_version_expiration = {
        newer_noncurrent_versions = 5
        days                      = 30
      }
    }
  ]
}

module "athena-s3-bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "3.15.1"

  bucket = "fleet-loadtesting-alb-logs-athena"

  # Allow deletion of non-empty bucket
  force_destroy = true

  attach_elb_log_delivery_policy        = true # Required for ALB logs
  attach_lb_log_delivery_policy         = true # Required for ALB/NLB logs
  attach_deny_insecure_transport_policy = true
  attach_require_latest_tls_policy      = true
  # attach_policy                         = var.extra_s3_athena_policies != []
  # policy                                = var.extra_s3_athena_policies != [] ? data.aws_iam_policy_document.s3_athena_bucket[0].json : null
  block_public_acls                     = true
  block_public_policy                   = true
  ignore_public_acls                    = true
  restrict_public_buckets               = true
  server_side_encryption_configuration = {
    rule = {
      bucket_key_enabled = true
      apply_server_side_encryption_by_default = {
        sse_algorithm = "AES256"
      }
    }
  }
  lifecycle_rule = [
    {
      id      = "log"
      enabled = true

      transition = [
        {
          days          = 90
          storage_class = "ONEZONE_IA"
        }
      ]
      expiration = {
        days = 365
        # Always resets to false anyhow showing terraform changes constantly
        expired_object_delete_marker = false
      }
      noncurrent_version_expiration = {
        newer_noncurrent_versions = 5
        days                      = 30
      }
    }
  ]
}

resource "aws_athena_database" "logs" {
  name   = "fleet_loadtesting_alb_logs"
  bucket = module.athena-s3-bucket.s3_bucket_id
}

resource "aws_athena_workgroup" "logs" {
  name  = "fleet-loadtesting-logs"

  configuration {
    enforce_workgroup_configuration    = true
    publish_cloudwatch_metrics_enabled = true

    result_configuration {
      output_location = "s3://${module.athena-s3-bucket.s3_bucket_id}/output/"

      encryption_configuration {
        encryption_option = "SSE_S3"
      }
    }
  }

  force_destroy = true
}

resource "aws_alb" "main" {
  name                       = "fleetdm"
  internal                   = false #tfsec:ignore:aws-elb-alb-not-public
  security_groups            = [aws_security_group.lb.id]
  subnets                    = module.vpc.public_subnets
  idle_timeout               = 905
  drop_invalid_header_fields = true
  #checkov:skip=CKV_AWS_150:don't like it

  access_logs {
    bucket  = module.s3_bucket_for_logs.s3_bucket_id
    prefix  = "alb-logs"
    enabled = true
  }

}

resource "aws_alb_listener" "https-fleetdm" {
  load_balancer_arn = aws_alb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate_validation.wildcard.certificate_arn

  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "moved to subdomains, try https://default.loadtest.fleetdm.com"
      status_code  = "404"
    }
  }
}

resource "aws_alb_listener" "http" {
  load_balancer_arn = aws_alb.main.arn
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

# Security group for the public internet facing load balancer
resource "aws_security_group" "lb" {
  name        = "${local.prefix} load balancer"
  description = "${local.prefix} Load balancer security group"
  vpc_id      = module.vpc.vpc_id
}

# Allow traffic from public internet
resource "aws_security_group_rule" "lb-ingress" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "443"
  to_port     = "443"
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-ingress-sgr

  security_group_id = aws_security_group.lb.id
}

resource "aws_security_group_rule" "lb-http-ingress" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "80"
  to_port     = "80"
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-ingress-sgr

  security_group_id = aws_security_group.lb.id
}

resource "aws_security_group_rule" "lb-es" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "9200"
  to_port     = "9200"
  protocol    = "tcp"
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = aws_security_group.lb.id
}
resource "aws_security_group_rule" "lb-es-apm" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "8200"
  to_port     = "8200"
  protocol    = "tcp"
  cidr_blocks = concat(["10.0.0.0/8"], [for ip in module.vpc.nat_public_ips : "${ip}/32"])

  security_group_id = aws_security_group.lb.id
}
resource "aws_security_group_rule" "lb-kibana" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "5601"
  to_port     = "5601"
  protocol    = "tcp"
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = aws_security_group.lb.id
}

# Allow outbound traffic
resource "aws_security_group_rule" "lb-egress" {
  description = "${local.prefix}: allow all outbound traffic"
  type        = "egress"

  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-egress-sgr

  security_group_id = aws_security_group.lb.id
}

