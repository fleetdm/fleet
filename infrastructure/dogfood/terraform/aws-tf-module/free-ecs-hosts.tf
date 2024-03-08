## Linux hosts in ECS

locals {
  osquery_hosts = {
    "5.8.2-ubuntu22.04@sha256:b77c7b06c4d7f2a3c58cc3a34e51fffc480e97795fb3c75cb1dc1cf3709e3dc6" = "Skys-laptop"
    "5.8.2-ubuntu20.04@sha256:3496ffd0ad570c88a9f405e6ef517079cfeed6ce405b9d22db4dc5ef6ed3faac" = "Cloud-City-server"
    "5.8.2-ubuntu18.04@sha256:372575e876c218dde3c5c0e24fd240d193800fca9b314e94b4ad4e6e22006c9b" = "Mists-laptop"
    "5.8.2-ubuntu16.04@sha256:112655c42951960d8858c116529fb4c64951e4cf2e34cb7c08cd599a009025bb" = "Ethers-laptop"
    "5.8.2-debian10@sha256:de29337896aac89b2b03c7642805859d3fb6d52e5dc08230f987bbab4eeba9c5"    = "Breezes-laptop"
    "5.8.2-debian9@sha256:47e46c19cebdf0dc704dd0061328856bda7e1e86b8c0fefdd6f78bd092c6200e"     = "Aero-server"
    "5.8.2-centos8@sha256:88a8adde80bd3b1b257e098bc6e41b6afea840f60033653dcb9fe984f36b0f97"     = "Stratuss-laptop"
    "5.8.2-centos7@sha256:ff251de4935b80a91c5fc1ac352aebdab9a6bbbf5bda1aaada8e26d22b50202d"     = "Zephyrs-Laptop"
    "5.8.2-centos6@sha256:b56736be8436288d3fbd2549ec6165e0588cd7197e91600de4a2f00f1df28617"     = "Halo-server"
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
  for_each    = local.osquery_hosts
  source      = "./docker"
  ecr_repo    = aws_ecr_repository.osquery.repository_url
  osquery_tag = each.key
}

resource "random_uuid" "osquery" {
  for_each = local.osquery_hosts
}

resource "aws_ecs_task_definition" "osquery" {
  for_each = local.osquery_hosts
  // e.g. 5-8-2-ubuntu22-04 to match naming requirements 
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
        image       = module.osquery_docker[each.key].ecr_image
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
  # Name must match ^[A-Za-z-_]+$ e.g. 5-8-2-ubuntu22-04
  name            = "osquery_${replace(split("@sha256", each.key)[0], ".", "-")}"
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
