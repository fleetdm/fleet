provider "aws" {
  default_tags {
    tags = {
      environment = "dogfood"
      terraform   = "https://github.com/fleetdm/fleet/main/infrastructure/dogfood/terraform"
      state       = "s3://fleet-terraform-remote-state/fleet"
    }
  }
}

terraform {
  // these values should match what is bootstrapped in ./remote-state
  backend "s3" {
    bucket         = "fleet-terraform-remote-state"
    region         = "us-east-2"
    key            = "fleet"
    dynamodb_table = "fleet-terraform-state-lock"
  }
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.0.2"
    }
  }
}

variable "fleet_license" {}
variable "fleet_image" {
  default = "160035666661.dkr.ecr.us-east-2.amazonaws.com/fleet:1f68e7a5e39339d763da26a0c8ae3e459b2e1f016538d7962312310493381f7c"
}
variable "geolite2_license" {}
variable "fleet_sentry_dsn" {}
variable "elastic_url" {}
variable "elastic_token" {}
variable "fleet_calendar_periodicity" {
  default     = "30s"
  description = "The refresh period for the calendar integration."
}
variable "dogfood_sidecar_enroll_secret" {}

data "aws_caller_identity" "current" {}

locals {
  customer       = "fleet-dogfood"
  fleet_image    = var.fleet_image # Set this to the version of fleet to be deployed
  geolite2_image = "${aws_ecr_repository.fleet.repository_url}:${split(":", var.fleet_image)[1]}-geolite2-${formatdate("YYYYMMDDhhmm", timestamp())}"
  extra_environment_variables = {
    FLEET_LICENSE_KEY                          = var.fleet_license
    FLEET_LOGGING_DEBUG                        = "true"
    FLEET_LOGGING_JSON                         = "true"
    FLEET_LOGGING_TRACING_ENABLED              = "true"
    FLEET_LOGGING_TRACING_TYPE                 = "elasticapm"
    FLEET_MYSQL_MAX_OPEN_CONNS                 = "25"
    FLEET_VULNERABILITIES_DATABASES_PATH       = "/home/fleet"
    FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING = "false"
    ELASTIC_APM_SERVER_URL                     = var.elastic_url
    ELASTIC_APM_SECRET_TOKEN                   = var.elastic_token
    ELASTIC_APM_SERVICE_NAME                   = "dogfood"
    FLEET_CALENDAR_PERIODICITY                 = var.fleet_calendar_periodicity
  }
  sentry_secrets = {
    FLEET_SENTRY_DSN = "${aws_secretsmanager_secret.sentry.arn}:FLEET_SENTRY_DSN::"
  }
  idp_metadata_file = "${path.module}/files/idp-metadata.xml"
}

module "main" {
  source          = "github.com/fleetdm/fleet//terraform?ref=tf-mod-root-v1.9.1"
  certificate_arn = module.acm.acm_certificate_arn
  vpc = {
    name = local.customer
  }
  rds_config = {
    name                = local.customer
    engine_version      = "8.0.mysql_aurora.3.07.1"
    snapshot_identifier = "arn:aws:rds:us-east-2:611884880216:cluster-snapshot:a2023-03-06-pre-migration"
    db_parameters = {
      # 8mb up from 262144 (256k) default
      sort_buffer_size = 8388608
    }
    # VPN
    allowed_cidr_blocks = ["10.255.1.0/24", "10.255.2.0/24", "10.255.3.0/24"]
  }
  redis_config = {
    name = local.customer
    log_delivery_configuration = [{
      destination      = "dogfood-redis-logs"
      destination_type = "cloudwatch-logs"
      log_format       = "json"
      log_type         = "engine-log"
    }]
  }
  ecs_cluster = {
    cluster_name = local.customer
  }
  fleet_config = {
    image    = local.geolite2_image
    family   = local.customer
    task_cpu = 1024
    task_mem = 4096
    cpu      = 1024
    mem      = 4096
    pid_mode = "task"
    autoscaling = {
      min_capacity = 2
      max_capacity = 5
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
    extra_iam_policies           = concat(module.firehose-logging.fleet_extra_iam_policies, module.osquery-carve.fleet_extra_iam_policies, module.ses.fleet_extra_iam_policies)
    extra_execution_iam_policies = concat(module.mdm.extra_execution_iam_policies, [aws_iam_policy.sentry.arn, aws_iam_policy.osquery_sidecar.arn]) #, module.saml_auth_proxy.fleet_extra_execution_policies)
    extra_environment_variables = merge(
      module.mdm.extra_environment_variables,
      module.firehose-logging.fleet_extra_environment_variables,
      module.osquery-carve.fleet_extra_environment_variables,
      module.ses.fleet_extra_environment_variables,
      local.extra_environment_variables,
      module.geolite2.extra_environment_variables,
      module.vuln-processing.extra_environment_variables
    )
    extra_secrets           = merge(module.mdm.extra_secrets, local.sentry_secrets)
    private_key_secret_name = "${local.customer}-fleet-server-private-key"
    # extra_load_balancers         = [{
    #   target_group_arn = module.saml_auth_proxy.lb_target_group_arn
    #   container_name   = "fleet"
    #   container_port   = 8080
    # }]
    software_installers = {
      bucket_prefix = "${local.customer}-software-installers-"
    }
    # sidecars = [
    #   {
    #     name        = "osquery"
    #     image       = module.osquery_docker.ecr_images["${local.osquery_version}-ubuntu24.04"]
    #     cpu         = 1024
    #     memory      = 1024
    #     mountPoints = []
    #     volumesFrom = []
    #     essential   = true
    #     ulimits = [
    #       {
    #         softLimit = 999999,
    #         hardLimit = 999999,
    #         name      = "nofile"
    #       }
    #     ]
    #     networkMode = "awsvpc"
    #     logConfiguration = {
    #       logDriver = "awslogs"
    #       options = {
    #         awslogs-group         = local.customer
    #         awslogs-region        = "us-east-2"
    #         awslogs-stream-prefix = "osquery"
    #       }
    #     }
    #     secrets = [
    #       {
    #         name      = "ENROLL_SECRET"
    #         valueFrom = aws_secretsmanager_secret.dogfood_sidecar_enroll_secret.arn
    #       }
    #     ]
    #     workingDirectory = "/",
    #     command = [
    #       "osqueryd",
    #       "--tls_hostname=dogfood.fleetdm.com",
    #       "--force=true",
    #       # Ensure that the host identifier remains the same between invocations
    #       # "--host_identifier=specified",
    #       # "--specified_identifier=${random_uuid.osquery[each.key].result}",
    #       "--verbose=true",
    #       "--tls_dump=true",
    #       "--enroll_secret_env=ENROLL_SECRET",
    #       "--enroll_tls_endpoint=/api/osquery/enroll",
    #       "--config_plugin=tls",
    #       "--config_tls_endpoint=/api/osquery/config",
    #       "--config_refresh=10",
    #       "--disable_distributed=false",
    #       "--distributed_plugin=tls",
    #       "--distributed_interval=10",
    #       "--distributed_tls_max_attempts=3",
    #       "--distributed_tls_read_endpoint=/api/osquery/distributed/read",
    #       "--distributed_tls_write_endpoint=/api/osquery/distributed/write",
    #       "--logger_plugin=tls",
    #       "--logger_tls_endpoint=/api/osquery/log",
    #       "--logger_tls_period=10",
    #       "--disable_carver=false",
    #       "--carver_start_endpoint=/api/osquery/carve/begin",
    #       "--carver_continue_endpoint=/api/osquery/carve/block",
    #       "--carver_block_size=8000000",
    #     ]
    #   }
    # ]
  }
  alb_config = {
    name = local.customer
    access_logs = {
      bucket  = module.logging_alb.log_s3_bucket_id
      prefix  = local.customer
      enabled = true
    }
    idle_timeout = 905
    #    extra_target_groups = [
    #      {
    #        name             = module.saml_auth_proxy.name
    #        backend_protocol = "HTTP"
    #        backend_port     = 80
    #        target_type      = "ip"
    #        health_check = {
    #          path                = "/_health"
    #          matcher             = "200"
    #          timeout             = 10
    #          interval            = 15
    #          healthy_threshold   = 5
    #          unhealthy_threshold = 5
    #        }
    #      }
    #    ]
    #    https_listener_rules = [{
    #      https_listener_index = 0
    #      priority             = 9000
    #      actions = [{
    #        type               = "forward"
    #        target_group_index = 1
    #      }]
    #      conditions = [{
    #        path_patterns = ["/device/*", "/api/*/fleet/device/*", "/saml/*"]
    #      }]
    #      }, {
    #      https_listener_index = 0
    #      priority             = 1
    #      actions = [{
    #        type               = "forward"
    #        target_group_index = 0
    #      }]
    #      conditions = [{
    #        path_patterns = ["/api/*/fleet/device/*/migrate_mdm", "/api/*/fleet/device/*/rotate_encryption_key"]
    #      }]
    #      }, {
    #      https_listener_index = 0
    #      priority             = 2
    #      actions = [{
    #        type               = "forward"
    #        target_group_index = 0
    #      }]
    #      conditions = [{
    #        path_patterns = ["/api/*/fleet/device/*/debug/errors", "/api/*/fleet/device/*/desktop"]
    #      }]
    #      }, {
    #      https_listener_index = 0
    #      priority             = 3
    #      actions = [{
    #        type               = "forward"
    #        target_group_index = 0
    #      }]
    #      conditions = [{
    #        path_patterns = ["/api/*/fleet/device/*/refetch", "/api/*/fleet/device/*/transparency"]
    #      }]
    #    }]
  }
}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = "dogfood.fleetdm.com"
  zone_id     = aws_route53_zone.main.id

  wait_for_validation = true
}

resource "aws_route53_zone" "main" {
  name = "dogfood.fleetdm.com"
}

resource "aws_route53_record" "main" {
  zone_id = aws_route53_zone.main.id
  name    = "dogfood.fleetdm.com"
  type    = "A"

  alias {
    name                   = module.main.byo-vpc.byo-db.alb.lb_dns_name
    zone_id                = module.main.byo-vpc.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

resource "aws_secretsmanager_secret" "sentry" {
  name = "${local.customer}-sentry"
}

resource "aws_secretsmanager_secret_version" "sentry" {
  secret_id = aws_secretsmanager_secret.sentry.id
  secret_string = jsonencode({
    FLEET_SENTRY_DSN = var.fleet_sentry_dsn
  })
}

resource "aws_iam_policy" "sentry" {
  name   = "fleet-sentry-secret-policy"
  policy = data.aws_iam_policy_document.sentry.json
}

data "aws_iam_policy_document" "sentry" {
  statement {
    actions = [
      "secretsmanager:GetSecretValue",
    ]
    resources = [aws_secretsmanager_secret.sentry.arn]
  }
}

module "migrations" {
  depends_on = [
    module.geolite2
  ]
  source                   = "github.com/fleetdm/fleet//terraform/addons/migrations?ref=tf-mod-addon-migrations-v2.0.1"
  ecs_cluster              = module.main.byo-vpc.byo-db.byo-ecs.service.cluster
  task_definition          = module.main.byo-vpc.byo-db.byo-ecs.task_definition.family
  task_definition_revision = module.main.byo-vpc.byo-db.byo-ecs.task_definition.revision
  subnets                  = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups          = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
  ecs_service              = module.main.byo-vpc.byo-db.byo-ecs.service.name
  desired_count            = module.main.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
  min_capacity             = module.main.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
  vuln_service             = module.vuln-processing.vuln_service_arn
}

module "mdm" {
  source             = "github.com/fleetdm/fleet//terraform/addons/mdm?ref=tf-mod-addon-mdm-v1.3.0"
  public_domain_name = "dogfood.fleetdm.com"
  enable_windows_mdm = true
  apn_secret_name    = "${local.customer}-apn"
  scep_secret_name   = "${local.customer}-scep"
  dep_secret_name    = "${local.customer}-dep"
}

module "firehose-logging" {
  source = "github.com/fleetdm/fleet//terraform/addons/logging-destination-firehose?ref=tf-mod-addon-logging-destination-firehose-v1.1.0"
  osquery_results_s3_bucket = {
    name = "${local.customer}-osquery-results-archive"
  }
  osquery_status_s3_bucket = {
    name = "${local.customer}-fleet-osquery-status-archive"
  }
}

module "osquery-carve" {
  source = "github.com/fleetdm/fleet//terraform/addons/osquery-carve?ref=tf-mod-addon-osquery-carve-v1.1.0"
  osquery_carve_s3_bucket = {
    name = "fleet-${local.customer}-osquery-carve"
  }
}

module "monitoring" {
  source                 = "github.com/fleetdm/fleet//terraform/addons/monitoring?ref=tf-mod-addon-monitoring-v1.5.0"
  customer_prefix        = local.customer
  fleet_ecs_service_name = module.main.byo-vpc.byo-db.byo-ecs.service.name
  albs = [
    {
      name                    = module.main.byo-vpc.byo-db.alb.lb_dns_name,
      target_group_name       = module.main.byo-vpc.byo-db.alb.target_group_names[0]
      target_group_arn_suffix = module.main.byo-vpc.byo-db.alb.target_group_arn_suffixes[0]
      arn_suffix              = module.main.byo-vpc.byo-db.alb.lb_arn_suffix
      ecs_service_name        = module.main.byo-vpc.byo-db.byo-ecs.service.name
      min_containers          = module.main.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
      alert_thresholds = {
        HTTPCode_ELB_5XX_Count = {
          period    = 3600
          threshold = 2
        },
        HTTPCode_Target_5XX_Count = {
          period    = 120
          threshold = 0
        }
      }
    }
  ]
  sns_topic_arns_map = {
    alb_httpcode_5xx            = [module.notify_slack.slack_topic_arn]
    cron_monitoring             = [module.notify_slack.slack_topic_arn]
    cron_job_failure_monitoring = [module.notify_slack_p2.slack_topic_arn]
  }
  mysql_cluster_members = module.main.byo-vpc.rds.cluster_members
  # The cloudposse module seems to have a nested list here.
  redis_cluster_members = module.main.byo-vpc.redis.member_clusters[0]
  acm_certificate_arn   = module.acm.acm_certificate_arn
  cron_monitoring = {
    mysql_host                 = module.main.byo-vpc.rds.cluster_reader_endpoint
    mysql_database             = module.main.byo-vpc.rds.cluster_database_name
    mysql_user                 = module.main.byo-vpc.rds.cluster_master_username
    mysql_password_secret_name = module.main.byo-vpc.secrets.secret_ids["${local.customer}-database-password"]
    rds_security_group_id      = module.main.byo-vpc.rds.security_group_id
    subnet_ids                 = module.main.vpc.private_subnets
    vpc_id                     = module.main.vpc.vpc_id
    # Format of https://pkg.go.dev/time#ParseDuration
    delay_tolerance = "4h"
    # Interval format for: https://docs.aws.amazon.com/scheduler/latest/UserGuide/schedule-types.html#rate-based
    run_interval = "1 hour"
  }
}

module "logging_alb" {
  source        = "github.com/fleetdm/fleet//terraform/addons/logging-alb?ref=tf-mod-addon-logging-alb-v1.2.0"
  prefix        = local.customer
  enable_athena = true
}

resource "aws_iam_policy" "ecr" {
  name   = "fleet-ecr-policy"
  policy = data.aws_iam_policy_document.ecr.json
}

data "aws_iam_policy_document" "ecr" {
  statement {
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:GetDownloadUrlForLayer",
      "ecr:GetAuthorizationToken"
    ]
    resources = ["*"]
  }
  statement {
    actions = [ #tfsec:ignore:aws-iam-no-policy-wildcards
      "kms:Encrypt*",
      "kms:Decrypt*",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:Describe*"
    ]
    resources = [aws_kms_key.ecr.arn]
  }
}

resource "aws_ecr_repository" "fleet" {
  name                 = "fleet"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.ecr.arn
  }
}

resource "aws_kms_key" "ecr" {
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

variable "slack_p1_webhook" {
  type = string
}

variable "slack_p2_webhook" {
  type = string
}

module "notify_slack" {
  source  = "terraform-aws-modules/notify-slack/aws"
  version = "5.5.0"

  sns_topic_name = "fleet-dogfood-p1-alerts"

  slack_webhook_url = var.slack_p1_webhook
  slack_channel     = "#help-p1"
  slack_username    = "monitoring"
}

module "notify_slack_p2" {
  source  = "terraform-aws-modules/notify-slack/aws"
  version = "5.5.0"

  sns_topic_name = "fleet-dogfood-p2-alerts"

  slack_webhook_url = var.slack_p2_webhook
  slack_channel     = "#help-p2"
  slack_username    = "monitoring"
}

module "ses" {
  source  = "github.com/fleetdm/fleet//terraform/addons/ses?ref=tf-mod-addon-ses-v1.0.0"
  zone_id = aws_route53_zone.main.zone_id
  domain  = "dogfood.fleetdm.com"
}

# module "saml_auth_proxy" {
#   # source                       = "github.com/fleetdm/fleet//terraform/addons/saml-auth-proxy?ref=main"
#   # public_alb_security_group_id = module.main.byo-vpc.byo-db.alb.security_group_id
#   idp_metadata_url             = "https://dev-99185346.okta.com/app/exkbcrjeqmahXWvW45d7/sso/saml/metadata"
#   customer_prefix              = local.customer
#   ecs_cluster                  = module.main.byo-vpc.byo-db.byo-ecs.service.cluster
#   ecs_execution_iam_role_arn   = module.main.byo-vpc.byo-db.byo-ecs.execution_iam_role_arn
#   ecs_iam_role_arn             = module.main.byo-vpc.byo-db.byo-ecs.iam_role_arn
#   security_groups              = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
#   base_url                     = "https://dogfood.fleetdm.com/"
#   subnets                      = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
#   vpc_id                       = module.main.vpc.vpc_id
#   logging_options              = null # Figure it out later
#   alb_target_group_arn         = module.main.byo-vpc.byo-db.alb.target_group_arns[1]
#   cookie_max_age               = "15m"
# }

# This is intended to be public
module "dogfood_idp_metadata_bucket" {
  source                                = "terraform-aws-modules/s3-bucket/aws"
  version                               = "3.15.1"
  bucket                                = "fleet-dogfood-idp-metadata"
  attach_deny_insecure_transport_policy = true
  attach_require_latest_tls_policy      = true
  attach_public_policy                  = true
  block_public_acls                     = false
  block_public_policy                   = false
  ignore_public_acls                    = false
  restrict_public_buckets               = false
  acl                                   = "public-read"
  control_object_ownership              = true
  object_ownership                      = "BucketOwnerPreferred"
}

resource "aws_s3_object" "idp_metadata" {
  bucket = module.dogfood_idp_metadata_bucket.s3_bucket_id
  key    = "idp-metadata.xml"
  source = local.idp_metadata_file
  etag   = filemd5(local.idp_metadata_file)
  acl    = "public-read"
}

module "geolite2" {
  source            = "github.com/fleetdm/fleet//terraform/addons/geolite2?ref=tf-mod-addon-geolite2-v1.0.0"
  fleet_image       = var.fleet_image
  destination_image = local.geolite2_image
  license_key       = var.geolite2_license
}

module "vuln-processing" {
  source                              = "github.com/fleetdm/fleet//terraform/addons/external-vuln-scans?ref=tf-mod-addon-external-vuln-scans-v2.2.0"
  ecs_cluster                         = module.main.byo-vpc.byo-db.byo-ecs.service.cluster
  execution_iam_role_arn              = module.main.byo-vpc.byo-db.byo-ecs.execution_iam_role_arn
  subnets                             = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups                     = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
  fleet_config                        = module.main.byo-vpc.byo-db.byo-ecs.fleet_config
  task_role_arn                       = module.main.byo-vpc.byo-db.byo-ecs.iam_role_arn
  fleet_server_private_key_secret_arn = module.main.byo-vpc.byo-db.byo-ecs.fleet_server_private_key_secret_arn
  awslogs_config = {
    group  = module.main.byo-vpc.byo-db.byo-ecs.fleet_config.awslogs.name
    region = module.main.byo-vpc.byo-db.byo-ecs.fleet_config.awslogs.region
    prefix = module.main.byo-vpc.byo-db.byo-ecs.fleet_config.awslogs.prefix
  }
  fleet_s3_software_installers_config = module.main.byo-vpc.byo-db.byo-ecs.fleet_s3_software_installers_config
}

resource "aws_secretsmanager_secret" "dogfood_sidecar_enroll_secret" {
  name = "dogfood-sidecar-enroll-secret"
}

resource "aws_secretsmanager_secret_version" "dogfood_sidecar_enroll_secret" {
  secret_id     = aws_secretsmanager_secret.dogfood_sidecar_enroll_secret.id
  secret_string = var.dogfood_sidecar_enroll_secret
}

data "aws_iam_policy_document" "osquery_sidecar" {
  statement {
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:GetDownloadUrlForLayer",
      "ecr:GetAuthorizationToken"
    ]
    resources = ["*"]
  }
  statement {
    actions = [ #tfsec:ignore:aws-iam-no-policy-wildcards
      "kms:Encrypt*",
      "kms:Decrypt*",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:Describe*"
    ]
    resources = [aws_kms_key.osquery.arn]
  }
  statement {
    actions = [ #tfsec:ignore:aws-iam-no-policy-wildcards
      "secretsmanager:GetSecretValue"
    ]
    resources = [aws_secretsmanager_secret.dogfood_sidecar_enroll_secret.arn]

  }
}

resource "aws_iam_policy" "osquery_sidecar" {
  name        = "osquery-sidecar-policy"
  description = "IAM policy that Osquery sidecar containers use to define access to AWS resources"
  policy      = data.aws_iam_policy_document.osquery_sidecar.json
}
