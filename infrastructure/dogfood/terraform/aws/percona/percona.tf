data "aws_ami" "percona" {
  most_recent = true

  filter {
    name   = "name"
    values = ["PMM2 Server *"]
  }

  owners = ["679593333241"] # Percona
}


resource "aws_route53_record" "record" {
  name    = "percona"
  type    = "A"
  zone_id = var.zone_id
  alias {
    evaluate_target_health = false
    name                   = aws_lb.main.dns_name
    zone_id                = aws_lb.main.zone_id
  }
}

resource "aws_lb" "main" {
  name = "percona"
  // Exposed to the Internet by design
  internal                   = false #tfsec:ignore:aws-elb-alb-not-public
  security_groups            = [aws_security_group.lb.id, aws_security_group.backend.id]
  subnets                    = var.public_subnets
  idle_timeout               = 905
  drop_invalid_header_fields = true
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate_validation.percona.certificate_arn

  default_action {
    target_group_arn = aws_lb_target_group.percona.arn
    type             = "forward"
  }
}

resource "aws_lb_target_group" "percona" {
  name        = "percona"
  protocol    = "HTTP"
  target_type = "instance"
  port        = "80"
  vpc_id      = var.vpc_id
}

resource "aws_lb_target_group_attachment" "percona" {
  target_group_arn = aws_lb_target_group.percona.arn
  target_id        = aws_instance.percona.id
}

// No permanent EBS encrypted volume at the moment.
resource "aws_instance" "percona" { #tfsec:ignore:aws-ec2-enable-at-rest-encryption:exp:2022-07-15
  ami                    = data.aws_ami.percona.id
  instance_type          = "m5.large"
  subnet_id              = var.private_subnet
  vpc_security_group_ids = [aws_security_group.backend.id]
  iam_instance_profile   = aws_iam_instance_profile.profile.name
  metadata_options {
    http_tokens = "required"
  }
}

resource "aws_iam_instance_profile" "profile" {
  name = "percona-profile"
  role = aws_iam_role.role.name
}

resource "aws_iam_role" "role" {
  name = "percona-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
// This bucket is single-purpose and using a wildcard is not problematic
resource "aws_iam_policy" "policy" { #tfsec:ignore:aws-iam-no-policy-wildcards
  name        = "percona-policy"
  description = "policy to discover rds instances"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Stmt1508404837000",
            "Effect": "Allow",
            "Action": [
                "rds:DescribeDBInstances",
                "cloudwatch:GetMetricStatistics",
                "cloudwatch:ListMetrics"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Sid": "Stmt1508410723001",
            "Effect": "Allow",
            "Action": [
                "logs:DescribeLogStreams",
                "logs:GetLogEvents",
                "logs:FilterLogEvents"
            ],
            "Resource": [
                "arn:aws:logs:*:*:log-group:RDSOSMetrics:*"
            ]
        }
    ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "test-attach" {
  role       = aws_iam_role.role.name
  policy_arn = aws_iam_policy.policy.arn
}
