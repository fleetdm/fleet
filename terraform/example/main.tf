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

locals {
  fleet_environment_variables = {
    # Uncomment and provide license key to unlock premium features.
    #      FLEET_LICENSE_KEY = "<enter_license_key>"
    # JSON logging improves the experience with Cloudwatch Log Insights
    FLEET_LOGGING_JSON                      = "true"
    FLEET_MYSQL_MAX_OPEN_CONNS              = "10"
    FLEET_MYSQL_READ_REPLICA_MAX_OPEN_CONNS = "10"
    # Vulnerabilities is a premium feature.
    # Uncomment as this is a writable location in the container.
    # FLEET_VULNERABILITIES_DATABASES_PATH    = "/home/fleet"
    FLEET_REDIS_MAX_OPEN_CONNS = "500"
    FLEET_REDIS_MAX_IDLE_CONNS = "500"
  }
}

module "fleet" {
  source          = "github.com/fleetdm/fleet//terraform?ref=tf-mod-root-v1.7.1"
  certificate_arn = module.acm.acm_certificate_arn

  vpc_config = {
    name = var.vpc_name
  }

  fleet_config = {
    # To avoid pull-rate limiting from dockerhub, consider using our quay.io mirror
    # for the Fleet image. e.g. "quay.io/fleetdm/fleet:v4.46.1"
    image = "fleetdm/fleet:v4.46.1" # override default to deploy the image you desire
    # See https://fleetdm.com/docs/deploy/reference-architectures#aws for appropriate scaling
    # memory and cpu.
    autoscaling = {
      min_capacity = 2
      max_capacity = 5
    }
    # 4GB Required for vulnerability scanning.  512MB works without.
    mem = 4096
    cpu = 512
    extra_environment_variables = merge(
      # Uncomment if enabling mdm module below.
      # module.mdm.extra_environment_variables,
      local.fleet_environment_variables
    )
    # Uncomment if enabling mdm module below.
    # extra_secrets = module.mdm.extra_secrets
    # extra_execution_iam_policies = module.mdm.extra_execution_iam_policies

  }
  rds_config = {
    # See https://fleetdm.com/docs/deploy/reference-architectures#aws for instance classes.
    instance_class = "db.t4g.medium"
  }
  redis_config = {
    # See https://fleetdm.com/docs/deploy/reference-architectures#aws for instance types.
    instance_type = "cache.t4g.small"
    # Note these parameters help performance with large/complex live queries.
    # See https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Troubleshooting-live-queries.md#1-redis for details.
    parameter = [
      { name = "client-output-buffer-limit-pubsub-hard-limit", value = 0 },
      { name = "client-output-buffer-limit-pubsub-soft-limit", value = 0 },
      { name = "client-output-buffer-limit-pubsub-soft-seconds", value = 0 },
    ]
  }
  alb_config = {
    # Script execution can run for up to 300s plus overhead.
    # Ensure the load balancer does not 5XX before we have results.
    idle_timeout = 305
  }
}

# Migrations will handle scaling Fleet to 0 running containers before running the DB migration task.
# This module will also handle scaling back up once migrations complete.
# NOTE: This requires the aws cli to be installed on the device running terraform as terraform
# doesn't directly support all the features required.  the aws cli is invoked via a null-resource.

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
