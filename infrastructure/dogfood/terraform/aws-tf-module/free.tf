locals {
  customer_free = "${local.customer}-free"
}

module "free" {
  source = "github.com/fleetdm/fleet//terraform/byo-vpc?ref=tf-mod-byo-vpc-v1.7.0"
  vpc_config = {
    name   = local.customer_free
    vpc_id = module.main.vpc.vpc_id
    networking = {
      subnets = module.main.vpc.private_subnets
    }
  }
  rds_config = {
    name                = local.customer_free
    snapshot_identifier = "arn:aws:rds:us-east-2:611884880216:cluster-snapshot:a2023-03-06-pre-migration"
    db_parameters = {
      # 8mb up from 262144 (256k) default
      sort_buffer_size = 8388608
    }
    # VPN
    allowed_cidr_blocks = ["10.255.1.0/24", "10.255.2.0/24", "10.255.3.0/24"]
    subnets             = module.main.vpc.database_subnets
  }
  redis_config = {
    name = local.customer_free
    log_delivery_configuration = [
      {
        destination      = "dogfood-free-redis-logs"
        destination_type = "cloudwatch-logs"
        log_format       = "json"
        log_type         = "engine-log"
      }
    ]
    subnets                       = module.main.vpc.elasticache_subnets
    elasticache_subnet_group_name = module.main.vpc.elasticache_subnet_group_name
    allowed_cidrs                 = module.main.vpc.private_subnets_cidr_blocks
    availability_zones            = ["us-east-2a", "us-east-2b", "us-east-2c"]
  }
  ecs_cluster = {
    cluster_name = local.customer_free
  }
  fleet_config = {
    image  = local.fleet_image
    family = local.customer_free
    awslogs = {
      name      = local.customer_free
      retention = 365
    }
    iam = {
      role = {
        name        = "${local.customer_free}-role"
        policy_name = "${local.customer_free}-iam-policy"
      }
      execution = {
        name        = "${local.customer_free}-execution-role"
        policy_name = "${local.customer_free}-iam-policy-execution"
      }
    }
    #    extra_iam_policies           = concat(module.firehose-logging.fleet_extra_iam_policies, module.osquery-carve.fleet_extra_iam_policies, module.ses.fleet_extra_iam_policies)
    #    extra_execution_iam_policies = concat(module.mdm.extra_execution_iam_policies, [aws_iam_policy.sentry.arn]) #, module.saml_auth_proxy.fleet_extra_execution_policies)
    #    extra_environment_variables  = merge(module.mdm.extra_environment_variables, module.firehose-logging.fleet_extra_environment_variables, module.osquery-carve.fleet_extra_environment_variables, module.ses.fleet_extra_environment_variables, local.extra_environment_variables)
    #    extra_secrets                = merge(module.mdm.extra_secrets, local.sentry_secrets)
    # extra_load_balancers         = [{
    #   target_group_arn = module.saml_auth_proxy.lb_target_group_arn
    #   container_name   = "fleet"
    #   container_port   = 8080
    # }]
  }
  alb_config = {
    name            = local.customer_free
    certificate_arn = module.acm-free.acm_certificate_arn
    subnets         = module.main.vpc.private_subnets
    access_logs = {
      bucket  = module.logging_alb.log_s3_bucket_id
      prefix  = local.customer_free
      enabled = true
    }
  }
}

module "acm-free" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = "free.fleetdm.com"
  zone_id     = aws_route53_zone.free.id

  wait_for_validation = true
}

resource "aws_route53_zone" "free" {
  name = "free.fleetdm.com"
}

resource "aws_route53_record" "free" {
  zone_id = aws_route53_zone.free.id
  name    = "free.fleetdm.com"
  type    = "A"

  alias {
    name                   = module.free.byo-db.alb.lb_dns_name
    zone_id                = module.free.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

module "ses-free" {
  source  = "github.com/fleetdm/fleet//terraform/addons/ses?ref=tf-mod-addon-ses-v1.0.0"
  zone_id = aws_route53_zone.free.zone_id
  domain  = "free.fleetdm.com"
}

module "waf-free" {
  source = "github.com/fleetdm/fleet//terraform/addons/waf-alb?ref=tf-mod-addon-waf-alb-v1.0.0"
  name   = local.customer_free
  lb_arn = module.free.byo-db.alb.lb_arn
}

#module "migrations" {
#  source                   = "github.com/fleetdm/fleet//terraform/addons/migrations?ref=tf-mod-addon-migrations-v1.0.0"
#  ecs_cluster              = module.free.byo-db.byo-ecs.service.cluster
#  task_definition          = module.free.byo-db.byo-ecs.task_definition.family
#  task_definition_revision = module.free.byo-db.byo-ecs.task_definition.revision
#  subnets                  = module.free.byo-db.byo-ecs.service.network_configuration[0].subnets
#  security_groups          = module.free.byo-db.byo-ecs.service.network_configuration[0].security_groups
#}