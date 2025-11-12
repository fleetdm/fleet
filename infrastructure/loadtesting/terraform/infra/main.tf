data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

data "git_repository" "tf" {
  directory = "${path.module}/../../../../"
}

data "aws_acm_certificate" "certificate" {
  domain      = "*.${data.aws_route53_zone.main.name}"
  statuses    = ["ISSUED"]
  types       = ["AMAZON_ISSUED"]
  most_recent = true
}

data "aws_route53_zone" "main" {
  name         = "loadtest.fleetdm.com."
  private_zone = false
}

resource "aws_route53_record" "main" {
  zone_id = data.aws_route53_zone.main.id
  name    = "${local.customer}.loadtest.fleetdm.com"
  type    = "A"

  alias {
    name                   = module.loadtest.byo-db.alb.lb_dns_name
    zone_id                = module.loadtest.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

module "loadtest" {
  source = "github.com/fleetdm/fleet-terraform//byo-vpc?ref=tf-mod-root-v1.18.3"
  vpc_config = {
    name   = local.customer
    vpc_id = data.terraform_remote_state.shared.outputs.vpc.vpc_id
    networking = {
      subnets = data.terraform_remote_state.shared.outputs.vpc.private_subnets
    }
  }
  rds_config = {
    name                         = local.customer
    instance_class               = var.database_instance_size
    replicas                     = var.database_instance_count
    engine_version               = "8.0.mysql_aurora.3.08.2"
    snapshot_identifier          = "arn:aws:rds:us-east-2:917007347864:cluster-snapshot:cleaned-8-0-teams-fixes-v4-55-0-minimum"
    preferred_maintenance_window = "fri:04:00-fri:05:00"
    # VPN
    subnets             = data.terraform_remote_state.shared.outputs.vpc.database_subnets
    allowed_cidr_blocks = concat(data.terraform_remote_state.shared.outputs.vpc.private_subnets_cidr_blocks, local.vpn_cidr_blocks)
    db_parameters = {
      # 8mb up from 262144 (256k) default
      sort_buffer_size = 8388608
    }
  }
  redis_config = {
    name                          = local.customer
    instance_type                 = var.redis_instance_size
    cluster_size                  = var.redis_instance_count
    subnets                       = data.terraform_remote_state.shared.outputs.vpc.private_subnets
    elasticache_subnet_group_name = data.terraform_remote_state.shared.outputs.vpc.elasticache_subnet_group_name
    allowed_cidrs                 = concat(data.terraform_remote_state.shared.outputs.vpc.private_subnets_cidr_blocks, local.vpn_cidr_blocks)
    # fleet-vpc has subnets in all 3 availability zones
    availability_zones            = ["us-east-2a", "us-east-2b", "us-east-2c"]
    parameter = [
      { name = "client-output-buffer-limit-pubsub-hard-limit", value = 0 },
      { name = "client-output-buffer-limit-pubsub-soft-limit", value = 0 },
      { name = "client-output-buffer-limit-pubsub-soft-seconds", value = 0 },
    ]
  }
  ecs_cluster = {
    cluster_name = local.customer
  }
  fleet_config = {
    image               = local.fleet_image
    family              = local.customer
    mem                 = var.fleet_task_memory
    cpu                 = var.fleet_task_cpu
    security_group_name = local.customer
    networking = {
      ingress_sources = {
        security_groups = [
          resource.aws_security_group.internal.id,
        ]
      }
    }
    extra_load_balancers = [{
      target_group_arn = resource.aws_lb_target_group.internal.arn
      container_name   = "fleet"
      container_port   = 8080
    }]
    autoscaling = {
      min_capacity = var.fleet_task_count
      max_capacity = var.fleet_task_count
    }
    awslogs = {
      name      = local.customer
      retention = 365
    }
    iam = {
      role = {
        name        = "${local.customer}-role"
        policy_name = "${local.customer}-iam-policy"
      }
      execution = {
        name        = "${local.customer}-execution-role"
        policy_name = "${local.customer}-iam-policy-execution"
      }
    }
    extra_iam_policies = concat(
      module.osquery-carve.fleet_extra_iam_policies,
      module.ses.fleet_extra_iam_policies,
      module.logging_firehose.fleet_extra_iam_policies,
    )
    # Add these for MDM or cloudfront
    extra_execution_iam_policies = concat(
      module.mdm.extra_execution_iam_policies,
      # module.cloudfront-software-installers.extra_execution_iam_policies,
      [
        resource.aws_iam_policy.license.arn
      ],
    )
    extra_environment_variables = merge(
      module.osquery-carve.fleet_extra_environment_variables,
      module.vuln-processing.extra_environment_variables,
      module.ses.fleet_extra_environment_variables,
      module.logging_firehose.fleet_extra_environment_variables,
      local.extra_environment_variables,
    )
    extra_secrets = merge(
      module.mdm.extra_secrets,
      # module.cloudfront-software-installers.extra_secrets,
      local.extra_secrets
    )
    private_key_secret_name = "${local.customer}-fleet-server-private-key"
    volumes = [
      {
        name = "rds-tls-certs"
      }
    ]
    mount_points = [
      {
        sourceVolume  = "rds-tls-certs",
        containerPath = local.rds_container_path
      }
    ]
    depends_on = [
      {
        containerName = "rds-tls-ca-retriever"
        condition     = "SUCCESS"
      },
      # {
      #   containerName = "prometheus-exporter"
      #   condition     = "START"
      # }
    ]
    sidecars = local.sidecars
  }
  alb_config = {
    name                       = local.customer
    enable_deletion_protection = false
    certificate_arn            = data.aws_acm_certificate.certificate.arn
    subnets                    = data.terraform_remote_state.shared.outputs.vpc.public_subnets
    access_logs = {
      bucket  = module.logging_alb.log_s3_bucket_id
      prefix  = local.customer
      enabled = true
    }
    idle_timeout = 905
  }
}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name         = "${local.customer}.loadtest.fleetdm.com"
  zone_id             = data.aws_route53_zone.main.id
  create_certificate  = false
  wait_for_validation = false
}

module "ses" {
  source            = "github.com/fleetdm/fleet-terraform//addons/ses?ref=tf-mod-addon-ses-v1.4.0"
  zone_id           = data.aws_route53_zone.main.id
  domain            = "${terraform.workspace}.loadtest.fleetdm.com"
  extra_txt_records = []
  custom_mail_from = {
    enabled       = true
    domain_prefix = "mail"
  }
}

module "migrations" {
  source                   = "github.com/fleetdm/fleet-terraform//addons/migrations?ref=tf-mod-addon-migrations-v2.1.0"
  ecs_cluster              = module.loadtest.byo-db.byo-ecs.service.cluster
  task_definition          = module.loadtest.byo-db.byo-ecs.task_definition.family
  task_definition_revision = module.loadtest.byo-db.byo-ecs.task_definition.revision
  subnets                  = module.loadtest.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups          = module.loadtest.byo-db.byo-ecs.service.network_configuration[0].security_groups
  ecs_service              = module.loadtest.byo-db.byo-ecs.service.name
  desired_count            = module.loadtest.byo-db.byo-ecs.appautoscaling_target.min_capacity
  min_capacity             = module.loadtest.byo-db.byo-ecs.appautoscaling_target.min_capacity

  depends_on = [
    module.loadtest,
    module.vuln-processing
  ]
}

module "vuln-processing" {
  source                              = "github.com/fleetdm/fleet-terraform//addons/external-vuln-scans?ref=tf-mod-addon-external-vuln-scans-v2.3.0"
  ecs_cluster                         = module.loadtest.byo-db.byo-ecs.service.cluster
  execution_iam_role_arn              = module.loadtest.byo-db.byo-ecs.execution_iam_role_arn
  subnets                             = module.loadtest.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups                     = module.loadtest.byo-db.byo-ecs.service.network_configuration[0].security_groups
  fleet_config                        = module.loadtest.byo-db.byo-ecs.fleet_config
  task_role_arn                       = module.loadtest.byo-db.byo-ecs.iam_role_arn
  fleet_server_private_key_secret_arn = module.loadtest.byo-db.byo-ecs.fleet_server_private_key_secret_arn
  awslogs_config = {
    group  = module.loadtest.byo-db.byo-ecs.fleet_config.awslogs.name
    region = module.loadtest.byo-db.byo-ecs.fleet_config.awslogs.region
    prefix = module.loadtest.byo-db.byo-ecs.fleet_config.awslogs.prefix
  }
  fleet_s3_software_installers_config = module.loadtest.byo-db.byo-ecs.fleet_s3_software_installers_config
}

module "mdm" {
  source             = "github.com/fleetdm/fleet-terraform/addons/mdm?depth=1&ref=tf-mod-addon-mdm-v2.0.0"
  apn_secret_name    = null
  scep_secret_name   = "${local.customer}-scep"
  abm_secret_name    = null
  enable_windows_mdm = true
  enable_apple_mdm   = false
}

module "osquery-carve" {
  source = "github.com/fleetdm/fleet-terraform//addons/osquery-carve?ref=tf-mod-addon-osquery-carve-v1.1.1"
  osquery_carve_s3_bucket = {
    name = "${local.customer}-osquery-carve"
  }
}

module "logging_alb" {
  source          = "github.com/fleetdm/fleet-terraform//addons/logging-alb?ref=tf-mod-addon-logging-alb-v1.6.1"
  prefix          = local.customer
  alt_path_prefix = local.customer
  enable_athena   = true
}

module "logging_firehose" {
  source = "github.com/fleetdm/fleet-terraform//addons/logging-destination-firehose?ref=tf-mod-addon-logging-destination-firehose-v1.2.4"
  prefix = local.customer
  osquery_results_s3_bucket = {
    name         = "${local.customer}-osquery-results-firehose-policy"
    expires_days = 1
  }
  osquery_status_s3_bucket = {
    name         = "${local.customer}-osquery-status-firehose-policy"
    expires_days = 1
  }
  audit_s3_bucket = {
    name         = "${local.customer}-audit-firehose-policy"
    expires_days = 1
  }
}
