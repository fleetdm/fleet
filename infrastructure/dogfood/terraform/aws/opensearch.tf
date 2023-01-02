resource "aws_security_group" "os" {
  name   = "dogfood"
  vpc_id = module.vpc.vpc_id

  ingress {
    from_port = 80
    to_port   = 443
    protocol  = "tcp"

    cidr_blocks = [
      "10.0.0.0/8" # TODO: vpn and dogfood SG only
    ]
  }
}

resource "aws_opensearch_domain" "main" {
  domain_name    = "dogfood"
  engine_version = "OpenSearch_1.3"

  cluster_config {
    instance_type          = "t3.small.search"
    instance_count         = 1
    zone_awareness_enabled = false
  }

  vpc_options {
    subnet_ids = module.vpc.private_subnets

    security_group_ids = [aws_security_group.os.id]
  }

  advanced_options = {
    "rest.action.multi.allow_explicit_index" = "true"
  }

  ebs_options {
    ebs_enabled = true
    volume_size = 10
    volume_type = "gp2"
  }

  access_policies = <<CONFIG
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "es:*",
            "Principal": "*",
            "Effect": "Allow",
            "Resource": "arn:aws:es:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:domain/dogfood/*"
        }
    ]
}
CONFIG
}

# data "aws_region" "current" {}
# data "aws_caller_identity" "current" {}

resource "aws_iam_service_linked_role" "main" {
  aws_service_name = "opensearchservice.amazonaws.com"
}
