terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.10.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1.2"
    }
    mysql = {
      source  = "petoju/mysql"
      version = "3.0.12"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "2.5.1"
    }
  }
  backend "s3" {}
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.cluster.endpoint
    token                  = data.aws_eks_cluster_auth.cluster.token
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  }
}

data "aws_eks_cluster" "cluster" {
  name = var.eks_cluster
}

data "aws_eks_cluster_auth" "cluster" {
  name = var.eks_cluster
}

provider "mysql" {
  endpoint = jsondecode(data.aws_secretsmanager_secret_version.mysql.secret_string)["endpoint"]
  username = jsondecode(data.aws_secretsmanager_secret_version.mysql.secret_string)["username"]
  password = jsondecode(data.aws_secretsmanager_secret_version.mysql.secret_string)["password"]
}

variable "mysql_secret" {}
variable "eks_cluster" {}
variable "redis_address" {}
variable "redis_database" {}
variable "lifecycle_table" {}
variable "base_domain" {}
variable "enroll_secret" {}
variable "installer_bucket" {}
variable "installer_bucket_arn" {}
variable "oidc_provider_arn" {}
variable "oidc_provider" {}
variable "kms_key_arn" {}
variable "ecr_url" {}
variable "license_key" {}
variable "apm_url" {}
variable "apm_token" {}

resource "mysql_user" "main" {
  user               = terraform.workspace
  host               = "%"
  plaintext_password = random_password.db.result
}

resource "mysql_database" "main" {
  name = terraform.workspace
}

resource "mysql_grant" "main" {
  user       = mysql_user.main.user
  database   = mysql_database.main.name
  host       = "%"
  privileges = ["ALL"]
}

data "aws_secretsmanager_secret_version" "mysql" {
  secret_id = var.mysql_secret
}

resource "random_password" "db" {
  length = 8
}

resource "random_integer" "cron_offset" {
  min = 0
  max = 14
}

resource "helm_release" "main" {
  name  = terraform.workspace
  chart = "${path.module}/fleet"

  set {
    name  = "fleetName"
    value = terraform.workspace
  }

  set {
    name  = "mysql.password"
    value = random_password.db.result
  }

  set {
    name  = "mysql.createSecret"
    value = true
  }

  set {
    name  = "mysql.secretName"
    value = terraform.workspace
  }

  set {
    name  = "mysql.username"
    value = mysql_user.main.user
  }

  set {
    name  = "mysql.database"
    value = terraform.workspace
  }

  set {
    name  = "mysql.address"
    value = jsondecode(data.aws_secretsmanager_secret_version.mysql.secret_string)["endpoint"]
  }

  set {
    name  = "fleet.tls.enabled"
    value = false
  }

  set {
    name  = "redis.address"
    value = var.redis_address
  }

  set {
    name  = "redis.database"
    value = var.redis_database
  }

  set {
    name  = "kubernetes.io/ingress.class"
    value = "nginx"
  }

  set {
    name  = "hostName"
    value = "${terraform.workspace}.${var.base_domain}"
  }

  set {
    name  = "ingressAnnotations.kubernetes\\.io/ingress\\.class"
    value = "haproxy"
  }

  set {
    name  = "replicas"
    value = "1"
  }

  set {
    name  = "imageTag"
    value = "v4.34.1"
  }

  set {
    name  = "imageRepo"
    value = var.ecr_url
  }

  set {
    name  = "packaging.enrollSecret"
    value = var.enroll_secret
  }

  set {
    name  = "packaging.s3.bucket"
    value = var.installer_bucket
  }

  set {
    name  = "packaging.s3.prefix"
    value = terraform.workspace
  }

  set {
    name  = "serviceAccountAnnotations.eks\\.amazonaws\\.com/role-arn"
    value = aws_iam_role.main.arn
  }

  set {
    name  = "crons.vulnerabilities"
    value = "${random_integer.cron_offset.result}\\,${random_integer.cron_offset.result + 15}\\,${random_integer.cron_offset.result + 30}\\,${random_integer.cron_offset.result + 45} 0,13-23 * * *"
  }

  set {
    name  = "fleet.licenseKey"
    value = var.license_key
  }

  set {
    name  = "apm.url"
    value = var.apm_url
  }

  set {
    name  = "apm.token"
    value = var.apm_token
  }
}

data "aws_iam_policy_document" "main" {
  statement {
    actions = [
      "s3:*Object",
      "s3:ListBucket",
    ]
    resources = [
      var.installer_bucket_arn,
      "${var.installer_bucket_arn}/${terraform.workspace}/*"
    ]
  }
  statement {
    actions = [
      "kms:DescribeKey",
      "kms:GenerateDataKey",
      "kms:Decrypt",
    ]
    resources = [var.kms_key_arn]
  }
}

resource "aws_iam_policy" "main" {
  name   = terraform.workspace
  policy = data.aws_iam_policy_document.main.json
}

resource "aws_iam_role_policy_attachment" "main" {
  role       = aws_iam_role.main.id
  policy_arn = aws_iam_policy.main.arn
}

data "aws_iam_policy_document" "main-assume-role" {
  statement {
    principals {
      type        = "Federated"
      identifiers = [var.oidc_provider_arn]
    }
    actions = ["sts:AssumeRoleWithWebIdentity"]
    condition {
      test     = "StringEquals"
      variable = "${var.oidc_provider}:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "${var.oidc_provider}:sub"
      values   = ["system:serviceaccount:default:${terraform.workspace}"]
    }
  }
}

resource "aws_iam_role" "main" {
  name_prefix        = terraform.workspace
  path               = "/sandbox/"
  assume_role_policy = data.aws_iam_policy_document.main-assume-role.json
}

resource "aws_dynamodb_table_item" "main" {
  table_name = var.lifecycle_table
  hash_key   = "ID"

  item = <<ITEM
{
  "ID": {"S": "${terraform.workspace}"},
  "State": {"S": "provisioned"},
  "redis_db": {"N": "${var.redis_database}"}
}
ITEM

  depends_on = [helm_release.main]
}
