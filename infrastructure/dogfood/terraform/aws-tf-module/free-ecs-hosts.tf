## Linux hosts in ECS

locals {
  osquery_version = "5.12.2"
  osquery_hosts = {
    "${local.osquery_version}-ubuntu24.04" = "Atmosphere-database"
    "${local.osquery_version}-ubuntu22.04" = "Skys-laptop"
    "${local.osquery_version}-ubuntu20.04" = "Cloud-City-server"
    "${local.osquery_version}-ubuntu18.04" = "Mists-laptop"
    "${local.osquery_version}-ubuntu16.04" = "Ethers-laptop"
    "${local.osquery_version}-debian10"    = "Breezes-laptop"
    "${local.osquery_version}-debian9"     = "Aero-server"
    "${local.osquery_version}-centos8"     = "Stratuss-laptop"
    "${local.osquery_version}-centos7"     = "Zephyrs-Laptop"
    "${local.osquery_version}-centos6"     = "Halo-server"
  }

}


# ECR to store the images
resource "aws_iam_role" "osquery" {
  name               = "fleet-free-osquery-execution"
  description        = "IAM Execution role for osquery containers"
  assume_role_policy = data.aws_iam_policy_document.osquery_assume_role.json
}

data "aws_iam_policy_document" "osquery_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["ecs.amazonaws.com", "ecs-tasks.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role_policy_attachment" "osquery_execution_attachment" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.osquery.name
}

resource "aws_iam_role_policy_attachment" "osquery" {
  policy_arn = aws_iam_policy.osquery.arn
  role       = aws_iam_role.osquery.name
}

resource "aws_iam_policy" "osquery" {
  name        = "osquery-ecr-policy"
  description = "IAM policy that Osquery containers use to define access to AWS resources"
  policy      = data.aws_iam_policy_document.osquery.json
}

data "aws_iam_policy_document" "osquery" {
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
    resources = [aws_secretsmanager_secret.osquery_enroll.arn]

  }
}

resource "aws_ecr_repository" "osquery" {
  name                 = "osquery"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.osquery.arn
  }
}

resource "aws_kms_key" "osquery" {
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_secretsmanager_secret" "osquery_enroll" {
  name = "osquery-enroll-secret"
}

output "osquery_repo" {
  value = aws_ecr_repository.osquery
}

output "osquery_iam_policy" {
  value = aws_iam_policy.osquery
}

data "aws_region" "current" {}
data "aws_ecr_authorization_token" "token" {}

provider "docker" {
  # Configuration options
  registry_auth {
    address  = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com"
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

module "osquery_docker" {
  source          = "./docker"
  ecr_repo        = aws_ecr_repository.osquery.repository_url
  osquery_version = local.osquery_version
  osquery_tags    = keys(local.osquery_hosts)
}

resource "random_uuid" "osquery" {
  for_each = local.osquery_hosts
}

resource "aws_ecs_task_definition" "osquery" {
  for_each = local.osquery_hosts
  // e.g. ${osquery_version}-ubuntu22-04 to match naming requirements
  family                   = "osquery-${replace(split("@sha256", each.key)[0], ".", "-")}"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.osquery.arn
  cpu                      = 256
  memory                   = 512
  # Needed to run hostname command
  container_definitions = jsonencode(
    [
      {
        name        = "osquery"
        image       = module.osquery_docker.ecr_images[each.key]
        cpu         = 256
        memory      = 512
        mountPoints = []
        volumesFrom = []
        essential   = true
        ulimits = [
          {
            softLimit = 999999,
            hardLimit = 999999,
            name      = "nofile"
          }
        ]
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options   = module.free.byo-db.byo-ecs.logging_config
        }
        environment = [
          {
            name  = "FAKE_HOSTNAME"
            value = each.value
          }
        ]
        secrets = [
          {
            name      = "ENROLL_SECRET"
            valueFrom = aws_secretsmanager_secret.osquery_enroll.arn
          }
        ]
        workingDirectory = "/",
        command = [
          "osqueryd",
          "--tls_hostname=free.fleetdm.com",
          "--force=true",
          # Ensure that the host identifier remains the same between invocations
          "--host_identifier=specified",
          "--specified_identifier=${random_uuid.osquery[each.key].result}",
          "--verbose=true",
          "--tls_dump=true",
          "--enroll_secret_env=ENROLL_SECRET",
          "--enroll_tls_endpoint=/api/osquery/enroll",
          "--config_plugin=tls",
          "--config_tls_endpoint=/api/osquery/config",
          "--config_refresh=10",
          "--disable_distributed=false",
          "--distributed_plugin=tls",
          "--distributed_interval=10",
          "--distributed_tls_max_attempts=3",
          "--distributed_tls_read_endpoint=/api/osquery/distributed/read",
          "--distributed_tls_write_endpoint=/api/osquery/distributed/write",
          "--logger_plugin=tls",
          "--logger_tls_endpoint=/api/osquery/log",
          "--logger_tls_period=10",
          "--disable_carver=false",
          "--carver_start_endpoint=/api/osquery/carve/begin",
          "--carver_continue_endpoint=/api/osquery/carve/block",
          "--carver_block_size=8000000",
        ]
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_ecs_service" "osquery" {
  for_each = local.osquery_hosts
  # Name must match ^[A-Za-z-_]+$ e.g. 5.12.2-ubuntu22-04
  name            = "osquery_${replace(each.key, ".", "-")}"
  launch_type     = "FARGATE"
  cluster         = module.free.byo-db.byo-ecs.service.cluster
  task_definition = aws_ecs_task_definition.osquery[each.key].arn
  desired_count   = 1
  # Spin down before spin up since we are specifying the host identifier manually
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100

  network_configuration {
    subnets         = module.free.byo-db.byo-ecs.service.network_configuration[0].subnets
    security_groups = module.free.byo-db.byo-ecs.service.network_configuration[0].security_groups
  }
}
