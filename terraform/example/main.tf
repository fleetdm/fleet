# This example doesn't cover using a remote backend for storing the current
# terraform state in S3 with a lock in DynamoDB (ideal for AWS) or other 
# methods. If using automation to apply the configuration or if multiple people
# will be managing these resources, this is recommended.
#
# See https://developer.hashicorp.com/terraform/language/settings/backends/s3
# for reference.

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.36.0"
    }
  }
}

locals {
  # Change these to match your environment.
  domain_name = "fleet.example.com"
  vpc_name = "fleet-vpc"
  # This creates a subdomain in AWS to manage DNS Records.
  # This allows for easy validation of TLS Certificates via ACM and
  # the use of alias records to the load balancer.  Please note if
  # this is a subdomain that NS records will be needed to be created
  # in the primary zone.  These NS records will be included in the outputs
  # of this terraform run.
  zone_name = "fleet.example.com"

  # Bucket names need to be unique across AWS.  Change this to a friendly
  # name to make finding carves in s3 easier later.
  osquery_carve_bucket_name   = "fleet-osquery-carve"
  osquery_results_bucket_name = "fleet-osquery-results"
  osquery_status_bucket_name  = "fleet-osquery-status"

  # Extra ENV Vars for Fleet customization can be set here.
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
  source          = "github.com/fleetdm/fleet//terraform?ref=tf-mod-root-v1.11.1"
  certificate_arn = module.acm.acm_certificate_arn

  vpc = {
    # By default, Availabililty zones for us-east-2 are configured. If an alternative region is desired,
    # configure the azs (3 required) variable below to the desired region.  If you have an exported AWS-REGION or a
    # region declared in ~/.aws/config, this value must match the region declared below.
    name = local.vpc_name
    # azs = ["ca-central-1a", "ca-central-1b", "ca-central-1d"]
  }

  fleet_config = {
    # To avoid pull-rate limiting from dockerhub, consider using our quay.io mirror
    # for the Fleet image. e.g. "quay.io/fleetdm/fleet:v4.60.0"
    image = "fleetdm/fleet:v4.60.0" # override default to deploy the image you desire
    # See https://fleetdm.com/docs/deploy/reference-architectures#aws for appropriate scaling
    # memory and cpu.
    autoscaling = {
      min_capacity = 2
      max_capacity = 5
    }
    # 4GB Required for vulnerability scanning.  512MB works without.
    mem = 4096
    cpu = 512
    extra_environment_variables = local.fleet_environment_variables
    # Uncomment if enabling mdm module below.
    # extra_secrets = module.mdm.extra_secrets
    # extra_execution_iam_policies = module.mdm.extra_execution_iam_policies
    extra_iam_policies = concat(
      module.osquery-carve.fleet_extra_iam_policies,
      module.firehose-logging.fleet_extra_iam_policies,
    )
  }
  rds_config = {
    # See https://fleetdm.com/docs/deploy/reference-architectures#aws for instance classes.
    instance_class = "db.t4g.medium"
    # Prevents edge case render failure in Audit log on the home screen.
    db_parameters = {
      # 8mb up from 262144 (256k) default
      sort_buffer_size = 8388608
    }
    # Uncomment to specify the RDS engine version
    # engine_version = "8.0.mysql_aurora.3.07.1"
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
    idle_timeout = 605
  }
}

# Migrations will handle scaling Fleet to 0 running containers before running the DB migration task.
# This module will also handle scaling back up once migrations complete.
# NOTE: This requires the aws cli to be installed on the device running terraform as terraform
# doesn't directly support all the features required.  the aws cli is invoked via a null-resource.

module "migrations" {
  source                   = "github.com/fleetdm/fleet//terraform/addons/migrations?ref=tf-mod-addon-migrations-v2.0.1"
  ecs_cluster              = module.fleet.byo-vpc.byo-db.byo-ecs.service.cluster
  task_definition          = module.fleet.byo-vpc.byo-db.byo-ecs.task_definition.family
  task_definition_revision = module.fleet.byo-vpc.byo-db.byo-ecs.task_definition.revision
  subnets                  = module.fleet.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].subnets
  security_groups          = module.fleet.byo-vpc.byo-db.byo-ecs.service.network_configuration[0].security_groups
  ecs_service              = module.fleet.byo-vpc.byo-db.byo-ecs.service.name
  desired_count            = module.fleet.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
  min_capacity             = module.fleet.byo-vpc.byo-db.byo-ecs.appautoscaling_target.min_capacity
}

module "osquery-carve" {
  # The carve bucket also stores software.
  source = "github.com/fleetdm/fleet//terraform/addons/osquery-carve?ref=tf-mod-addon-osquery-carve-v1.1.0"
  osquery_carve_s3_bucket = {
    name = local.osquery_carve_bucket_name
  }   
} 

module "firehose-logging" {
  source = "github.com/fleetdm/fleet//terraform/addons/logging-destination-firehose?ref=tf-mod-addon-logging-destination-firehose-v1.1.0"
  osquery_results_s3_bucket = {
    name = local.osquery_results_bucket_name
  }
  osquery_status_s3_bucket = {
    name = local.osquery_status_bucket_name
  }
}

## MDM

# MDM Secrets must be populated with JSON data including the payload from the certs, keys, challenge, etc.
# These can be populated via terraform with a secret-version, or manually after terraform is applied.
# Note: Services will not start if the mdm module is enabled and the secrets are applied but not populated.


## MDM Secret payload

# See https://github.com/fleetdm/fleet/blob/tf-mod-addon-mdm-v2.0.0/terraform/addons/mdm/README.md#abm
# Per that document, both Windows and Mac will use the same SCEP secret under the hood.


# module "mdm" {
#   source             = "github.com/fleetdm/fleet//terraform/addons/mdm?ref=tf-mod-addon-mdm-v2.0.0"
#   # Set apn_secret_name = null if not using mac mdm
#   apn_secret_name    = "fleet-apn"
#   scep_secret_name   = "fleet-scep"
#   # Set abm_secret_name = null if customer is not using dep
#   abm_secret_name    = "fleet-dep"
#   enable_apple_mdm   = true
#   enable_windows_mdm = true
# }

# If you want to supply the MDM secrets via terraform, I recommend that you do not store the secrets in the clear
# on the device that applies the terraform.  For the example here, terraform will create a KMS key, which will then
# be used to encrypt the secrets. The included mdm-secrets.tf file will then use the KMS key to dercrypt the secrets
# on the filesystem to generate the 

# resource "aws_kms_key" "fleet_data_key" {
#   description = "key used to encrypt sensitive data stored in terraform"
# }
#
# resource "aws_kms_alias" "alias" {
#   name          = "alias/fleet-terraform-encrypted"
#   target_key_id = aws_kms_key.fleet_data_key.id
# }
#
# output "kms_key_id" {
#   value = aws_kms_key.fleet_data_key.id
# }

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = local.domain_name
  # If you change the route53 zone to a data source this needs to become "data.aws_route53_zone.main.id"
  zone_id     = aws_route53_zone.main.id

  wait_for_validation = true
}

# If you already are managing your zone in AWS in the same account,
# this resource could be swapped with a data source instead to
# read the properties of that resource.
resource "aws_route53_zone" "main" {
  name = local.zone_name
}

resource "aws_route53_record" "main" {
  # If you change the route53_zone to a data source this also needs to become "data.aws_route53_zone.main.id"
  zone_id = aws_route53_zone.main.id
  name    = local.domain_name
  type    = "A"

  alias {
    name                   = module.fleet.byo-vpc.byo-db.alb.lb_dns_name
    zone_id                = module.fleet.byo-vpc.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

# Ensure that these records are added to the parent DNS zone
# Delete this output if you switched the route53 zone above to a data source.
output "route53_name_servers" {
  value = aws_route53_zone.main.name_servers
}
