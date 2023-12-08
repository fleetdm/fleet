#### Linux hosts in ECS

locals {
  osquery_tags = [
    "5.8.2-ubuntu22.04@sha256:b77c7b06c4d7f2a3c58cc3a34e51fffc480e97795fb3c75cb1dc1cf3709e3dc6",
    "5.8.2-ubuntu20.04@sha256:3496ffd0ad570c88a9f405e6ef517079cfeed6ce405b9d22db4dc5ef6ed3faac",
    "5.8.2-ubuntu18.04@sha256:372575e876c218dde3c5c0e24fd240d193800fca9b314e94b4ad4e6e22006c9b",
    "5.8.2-ubuntu16.04@sha256:112655c42951960d8858c116529fb4c64951e4cf2e34cb7c08cd599a009025bb",
    "5.8.2-debian10@sha256:de29337896aac89b2b03c7642805859d3fb6d52e5dc08230f987bbab4eeba9c5",
    "5.8.2-debian9@sha256:47e46c19cebdf0dc704dd0061328856bda7e1e86b8c0fefdd6f78bd092c6200e",
    "5.8.2-centos8@sha256:88a8adde80bd3b1b257e098bc6e41b6afea840f60033653dcb9fe984f36b0f97",
    "5.8.2-centos7@sha256:ff251de4935b80a91c5fc1ac352aebdab9a6bbbf5bda1aaada8e26d22b50202d",
    "5.8.2-centos6@sha256:b56736be8436288d3fbd2549ec6165e0588cd7197e91600de4a2f00f1df28617",
  ]

}


# ECR to store the images

resource "aws_iam_policy" "osquery_ecr" {
  name   = "osquery-ecr-policy"
  policy = data.aws_iam_policy_document.osquery_ecr.json
}

data "aws_iam_policy_document" "osquery_ecr" {
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
    resources = [aws_kms_key.osquery_ecr.arn]
  }
}

resource "aws_ecr_repository" "osquery" {
  name                 = "fleet"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.osquery_ecr.arn
  }
}

resource "aws_kms_key" "osquery_ecr" {
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

output "osquery_ecr_repo" {
  value = aws_ecr_repository.osquery
}

output "osquery_ecr_iam_policy" {
  value = aws_iam_policy.osquery_ecr
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
  for_each    = toset(local.osquery_tags)
  source      = "./docker"
  ecr_repo    = aws_ecr_repository.osquery.repository_url
  osquery_tag = each.key
}
