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
    host                   = var.cluster_endpoint
    cluster_ca_certificate = base64decode(var.cluster_ca_cert)
    token                  = data.aws_eks_cluster_auth.cluster.token
  }
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
variable "cluster_endpoint" {}
variable "cluster_ca_cert" {}

resource "mysql_user" "main" {
  user               = random_string.db.id
  plaintext_password = random_password.db.id
}

resource "mysql_database" "main" {
  name = random_pet.main.id
}

resource "mysql_grant" "main" {
  user       = mysql_user.main.user
  database   = mysql_database.main.name
  privileges = ["ALL"]
}

data "aws_secretsmanager_secret_version" "mysql" {
  secret_id = var.mysql_secret
}

resource "random_pet" "main" {
  length = 3
}

resource "random_password" "db" {
  length = 24
}

resource "random_string" "db" {
  length  = 24
  special = false
}

resource "helm_release" "main" {
  name       = random_pet.main.id
  repository = "todo" # TODO
  chart      = "todo" # TODO
  version    = "todo" # TODO

  set {
    name  = "fleetName"
    value = random_pet.main.id
  }

  set {
    name  = "createNamespace"
    value = false
  }

  set {
    name  = "mysql.password"
    value = random_password.db.id
  }

  set {
    name  = "mysql.username"
    value = random_string.db.id
  }

  set {
    name  = "mysql.endpoint"
    value = jsondecode(data.aws_secretsmanager_secret_version.mysql.secret_string)["endpoint"]
  }
}
