terraform {
  required_version = ">= 1.3.8"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
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

locals {
  fleet_image = "fleetdm/fleet:v4.62.3"
  domain_name = "example.com"
}

resource "random_pet" "main" {}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = "${random_pet.main.id}.${local.domain_name}"
  zone_id     = data.aws_route53_zone.main.id

  wait_for_validation = true
}

resource "aws_route53_record" "main" {
  zone_id = data.aws_route53_zone.main.id
  name    = "${random_pet.main.id}.${local.domain_name}"
  type    = "A"

  alias {
    name                   = module.byo-vpc.byo-db.alb.lb_dns_name
    zone_id                = module.byo-vpc.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

data "aws_route53_zone" "main" {
  name         = "${local.domain_name}."
  private_zone = false
}

module "firehose-logging" {
  source = "github.com/fleetdm/fleet//terraform/addons/logging-destination-firehose?ref=tf-mod-addon-logging-destination-firehose-v1.1.0"
  osquery_results_s3_bucket = {
    name = "${random_pet.main.id}-results"
  }
  osquery_status_s3_bucket = {
    name = "${random_pet.main.id}-status"
  }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.1.2"

  name = random_pet.main.id
  cidr = "10.10.0.0/16"

  azs                                       = ["us-east-2a", "us-east-2b", "us-east-2c"]
  private_subnets                           = ["10.10.1.0/24", "10.10.2.0/24", "10.10.3.0/24"]
  public_subnets                            = ["10.10.11.0/24", "10.10.12.0/24", "10.10.13.0/24"]
  database_subnets                          = ["10.10.21.0/24", "10.10.22.0/24", "10.10.23.0/24"]
  elasticache_subnets                       = ["10.10.31.0/24", "10.10.32.0/24", "10.10.33.0/24"]
  create_database_subnet_group              = false
  create_database_subnet_route_table        = true
  create_elasticache_subnet_group           = true
  create_elasticache_subnet_route_table     = true
  enable_vpn_gateway                        = false
  one_nat_gateway_per_az                    = false
  single_nat_gateway                        = true
  enable_nat_gateway                        = true
  enable_flow_log                           = false
  create_flow_log_cloudwatch_log_group      = false
  create_flow_log_cloudwatch_iam_role       = false
  flow_log_max_aggregation_interval         = null
  flow_log_cloudwatch_log_group_name_prefix = null
  flow_log_cloudwatch_log_group_name_suffix = null
  vpc_flow_log_tags                         = {}
  enable_dns_hostnames                      = false
  enable_dns_support                        = true
}

module "byo-vpc" {
  source = "github.com/fleetdm/fleet//terraform/byo-vpc?ref=tf-mod-byo-vpc-v1.10.1"
  vpc_config = {
    vpc_id = module.vpc.vpc_id
    networking = {
      subnets = module.vpc.private_subnets
    }
  }
  rds_config = {
    name           = random_pet.main.id
    instance_class = "db.t4g.large"
    subnets        = module.vpc.database_subnets
  }
  redis_config = {
    instance_size                 = "cache.m6g.large"
    subnets                       = module.vpc.elasticache_subnets
    elasticache_subnet_group_name = module.vpc.elasticache_subnet_group_name
    availability_zones            = module.vpc.azs
    allowed_cidrs                 = module.vpc.private_subnets_cidr_blocks
  }
  alb_config = {
    subnets         = module.vpc.public_subnets
    certificate_arn = module.acm.acm_certificate_arn
    https_listener_rules = [{
      https_listener_index = 0
      actions = [{
        type         = "fixed-response"
        content_type = "text/plain"
        status_code  = "200"
        message_body = "This message is delivered instead of Fleet."

      }]
      conditions = [{
        http_headers = [{
          http_header_name = "X-Fixed-Response"
          values           = ["yes", "true"]
        }]
      }]
    }]
  }
  ecs_cluster = {
    cluster_name = random_pet.main.id
  }
  fleet_config = {
    image = local.fleet_image
    cpu   = 512
    autoscaling = {
      min_capacity = 2
      max_capacity = 5
    }
    extra_secrets = {
      // FLEET_LICENSE_KEY: "secret_manager_license_key_arn" 
    }
    extra_environment_variables = module.firehose-logging.fleet_extra_environment_variables
    extra_iam_policies          = module.firehose-logging.fleet_extra_iam_policies
  }
}

module "migrations" {
  source                   = "github.com/fleetdm/fleet//terraform/addons/migrations?ref=tf-mod-addon-migrations-v1.0.0"
  ecs_cluster              = module.byo-vpc.byo-db.byo-ecs.service.cluster
  task_definition          = module.byo-vpc.byo-db.byo-ecs.task_definition.family
  task_definition_revision = module.byo-vpc.byo-db.byo-ecs.task_definition.revision
  subnets                  = module.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups          = module.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
}

