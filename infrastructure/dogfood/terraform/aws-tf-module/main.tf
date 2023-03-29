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
      version = "4.57.0"
    }
  }
}

variable "fleet_license" {}
variable "fleet_image" {
  default = "160035666661.dkr.ecr.us-east-2.amazonaws.com/fleet:1f68e7a5e39339d763da26a0c8ae3e459b2e1f016538d7962312310493381f7c"
}
variable "sentry_dsn" {
}

data "aws_caller_identity" "current" {}

locals {
  customer    = "fleet-dogfood"
  fleet_image = var.fleet_image # Set this to the version of fleet to be deployed
  extra_environment_variables = {
    FLEET_LICENSE_KEY                          = var.fleet_license
    FLEET_LOGGING_DEBUG                        = "true"
    FLEET_LOGGING_JSON                         = "true"
    FLEET_MYSQL_MAX_OPEN_CONNS                 = "25"
    FLEET_VULNERABILITIES_DATABASES_PATH       = "/home/fleet"
    FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING = "false"
  }
  sentry_secrets = {
    SENTRY_DSN = "${aws_secretsmanager_secret.sentry.arn}:SENTRY_DSN::"
  }
}

module "main" {
  source          = "github.com/fleetdm/fleet//terraform?ref=main"
  certificate_arn = module.acm.acm_certificate_arn
  vpc = {
    name = local.customer
  }
  rds_config = {
    name                = local.customer
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
  }
  ecs_cluster = {
    cluster_name = local.customer
  }
  fleet_config = {
    image  = local.fleet_image
    family = local.customer
    awslogs = {
      name = local.customer
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
    extra_execution_iam_policies = concat(module.mdm.extra_execution_iam_policies, [aws_iam_policy.sentry.arn])
    extra_environment_variables  = merge(module.mdm.extra_environment_variables, module.firehose-logging.fleet_extra_environment_variables, module.osquery-carve.fleet_extra_environment_variables, module.ses.fleet_extra_environment_variables, local.extra_environment_variables)
    extra_secrets                = merge(module.mdm.extra_secrets, local.sentry_secrets)
  }
  alb_config = {
    name = local.customer
    access_logs = {
      bucket  = module.logging_alb.log_s3_bucket_id
      prefix  = local.customer
      enabled = true
    }
    allowed_cidrs = [
      "128.0.0.0/1",
      "64.0.0.0/2",
      "0.0.0.0/3",
      "48.0.0.0/4",
      "40.0.0.0/5",
      "36.0.0.0/6",
      "32.0.0.0/7",
      "34.0.0.0/9",
      "34.128.0.0/10",
      "34.224.0.0/11",
      "34.192.0.0/12",
      "34.208.0.0/13",
      "34.216.0.0/14",
      "34.220.0.0/15",
      "34.222.0.0/16",
      "35.0.0.0/8",
    ]
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
    SENTRY_DSN = var.sentry_dsn
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
  source                   = "github.com/fleetdm/fleet//terraform/addons/migrations?ref=main"
  ecs_cluster              = module.main.byo-vpc.byo-db.byo-ecs.service.cluster
  task_definition          = module.main.byo-vpc.byo-db.byo-ecs.task_definition.family
  task_definition_revision = module.main.byo-vpc.byo-db.byo-ecs.task_definition.revision
  subnets                  = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups          = module.main.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
}

module "mdm" {
  source             = "github.com/fleetdm/fleet//terraform/addons/mdm?ref=main"
  public_domain_name = "dogfood.fleetdm.com"
  apn_secret_name    = "${local.customer}-apn"
  scep_secret_name   = "${local.customer}-scep"
  dep_secret_name    = "${local.customer}-dep"
}

module "firehose-logging" {
  source = "github.com/fleetdm/fleet//terraform/addons/logging-destination-firehose?ref=main"
  osquery_results_s3_bucket = {
    name = "${local.customer}-osquery-results-archive"
  }
  osquery_status_s3_bucket = {
    name = "${local.customer}-fleet-osquery-status-archive"
  }
}

module "osquery-carve" {
  source = "github.com/fleetdm/fleet//terraform/addons/osquery-carve?ref=main"
  osquery_carve_s3_bucket = {
    name = "${local.customer}-osquery-carve"
  }
}

module "monitoring" {
  source                      = "github.com/fleetdm/fleet//terraform/addons/monitoring?ref=main"
  customer_prefix             = local.customer
  fleet_ecs_service_name      = module.main.byo-vpc.byo-db.byo-ecs.service.name
  fleet_min_containers        = module.main.byo-vpc.byo-db.byo-ecs.service.desired_count
  alb_name                    = module.main.byo-vpc.byo-db.alb.lb_dns_name
  alb_target_group_name       = module.main.byo-vpc.byo-db.alb.target_group_names[0]
  alb_target_group_arn_suffix = module.main.byo-vpc.byo-db.alb.target_group_arn_suffixes[0]
  alb_arn_suffix              = module.main.byo-vpc.byo-db.alb.lb_arn_suffix
  sns_topic_arns_map = {
    alb_httpcode_5xx = [module.notify_slack.slack_topic_arn]
  }
  mysql_cluster_members = module.main.byo-vpc.rds.cluster_members
  # The cloudposse module seems to have a nested list here.
  redis_cluster_members = module.main.byo-vpc.redis.member_clusters[0]
  acm_certificate_arn   = module.acm.acm_certificate_arn
}

module "logging_alb" {
  source        = "github.com/fleetdm/fleet//terraform/addons/logging-alb?ref=main"
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

variable "slack_webhook" {
  type = string
}

module "notify_slack" {
  source  = "terraform-aws-modules/notify-slack/aws"
  version = "5.5.0"

  sns_topic_name = "fleet-dogfood"

  slack_webhook_url = var.slack_webhook
  slack_channel     = "#help-p1"
  slack_username    = "monitoring"
}

module "ses" {
  source = "github.com/fleetdm/fleet//terraform/addons/ses?ref=main"
  domain = "dogfood.fleetdm.com"
}
