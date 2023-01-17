terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  default_tags {
    tags = {
      Example = "This is a demo of the Fleet terraform module"
    }
  }
}

resource "random_pet" "main" {}

module "main" {
  source          = "../"
  certificate_arn = module.acm.acm_certificate_arn
  vpc = {
    name = random_pet.main.id
  }
  fleet_config = {
    extra_environment_variables = module.firehose-logging.fleet_extra_environment_variables
    extra_iam_policies          = module.firehose-logging.fleet_extra_iam_policies
  }
}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = "${random_pet.main.id}.loadtest.fleetdm.com"
  zone_id     = data.aws_route53_zone.main.id

  wait_for_validation = true
}

resource "aws_route53_record" "main" {
  zone_id = data.aws_route53_zone.main.id
  name    = "${random_pet.main.id}.loadtest.fleetdm.com"
  type    = "A"

  alias {
    name                   = module.main.byo-vpc.byo-db.alb.lb_dns_name
    zone_id                = module.main.byo-vpc.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

data "aws_route53_zone" "main" {
  name         = "loadtest.fleetdm.com."
  private_zone = false
}

module "firehose-logging" {
  source = "../addons/logging-destination-firehose"
  osquery_results_s3_bucket = {
    name = "${random_pet.main.id}-results"
  }
  osquery_status_s3_bucket = {
    name = "${random_pet.main.id}-status"
  }
}
