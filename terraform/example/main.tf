terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.36.0"
    }
  }
}

variable "domain_name" {
  type        = string
  description = "domain name to host fleet under"
}

variable "vpc_name" {
  type        = string
  description = "name of the vpc to provision"
  default     = "fleet"
}

variable "zone_name" {
  type        = string
  description = "the name to give to your hosted zone"
  default     = "fleet"
}

module "fleet" {
  source          = "github.com/fleetdm/fleet//terraform?ref=tf-mod-root-v1.7.1"
  certificate_arn = module.acm.acm_certificate_arn

  vpc_config = {
    name = var.vpc_name
  }

  fleet_config = {
    image = "fleetdm/fleet:v4.46.1" # override default to deploy the image you desire
    extra_environment_variables = {
      #      FLEET_LICENSE_KEY = "<enter_license_key>"
    }
  }
}

module "migrations" {
  source                   = "github.com/fleetdm/fleet//terraform/addons/migrations?ref=tf-mod-addon-migrations-v2.0.0"
  ecs_cluster              = module.fleet.byo-vpc.byo-db.byo-ecs.service.cluster
  task_definition          = module.fleet.byo-vpc.byo-db.byo-ecs.task_definition.family
  task_definition_revision = module.fleet.byo-vpc.byo-db.byo-ecs.task_definition.revision
  subnets                  = module.fleet.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups          = module.fleet.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
  ecs_service              = module.fleet.byo-vpc.byo-db.byo-ecs.service.name
  desired_count            = module.fleet.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
  min_capacity             = module.fleet.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = var.domain_name
  zone_id     = aws_route53_zone.main.id

  wait_for_validation = true
}

resource "aws_route53_zone" "main" {
  name = var.zone_name
}

resource "aws_route53_record" "main" {
  zone_id = aws_route53_zone.main.id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = module.fleet.byo-vpc.byo-db.alb.lb_dns_name
    zone_id                = module.fleet.byo-vpc.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}