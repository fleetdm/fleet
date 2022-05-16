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

resource "mysql_user" "main" {
  user               = random_string.main.id
  plaintext_password = random_password.db.id
}

resource "mysql_database" "main" {
  name = random_string.main.id
}

resource "mysql_grant" "main" {
  user       = mysql_user.main.user
  database   = mysql_database.main.name
  privileges = ["ALL"]
}

data "aws_secretsmanager_secret_version" "mysql" {
  secret_id = var.mysql_secret
}

resource "random_password" "db" {
  length = 24
}

resource "random_string" "main" {
  length  = 10
  special = false
  upper   = false
  number  = false
}

resource "helm_release" "main" {
  name  = random_string.main.id
  chart = "${path.module}/fleet"

  set {
    name  = "fleetName"
    value = random_string.main.id
  }

  set {
    name  = "mysql.password"
    value = mysql_user.main.plaintext_password
  }

  set {
    name  = "mysql.createSecret"
    value = true
  }

  set {
    name  = "mysql.secretName"
    value = random_string.main.id
  }

  set {
    name  = "mysql.username"
    value = mysql_user.main.user
  }

  set {
    name  = "mysql.database"
    value = random_string.main.id
  }

  set {
    name  = "mysql.address"
    value = jsondecode(data.aws_secretsmanager_secret_version.mysql.secret_string)["endpoint"]
  }
}
