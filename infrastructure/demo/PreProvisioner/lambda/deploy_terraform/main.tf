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
}
