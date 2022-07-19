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

data "aws_secretsmanager_secret_version" "mysql" {
  secret_id = var.mysql_secret
}
