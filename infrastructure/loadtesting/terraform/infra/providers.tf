terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.68.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.16.0"
    }
    git = {
      source  = "paultyng/git"
      version = "~> 0.1.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "loadtesting"                               # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

provider "aws" {
  region = "us-east-2"
  default_tags {
    tags = {
      environment = "loadtest"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting"
      state       = "s3://fleet-terraform-state20220408141538466600000002/loadtesting/${terraform.workspace}/loadtesting/loadtesting/terraform.tfstate"
      workspace   = "${terraform.workspace}"
    }
  }
}

data "terraform_remote_state" "shared" {
  backend = "s3"
  config = {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/shared/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "loadtesting"                                      # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

data "terraform_remote_state" "eks_vpc" {
  count   = var.enable_otel ? 1 : 0
  backend = "s3"
  config = {
    bucket         = "fleet-terraform-state20220408141538466600000002"
    key            = "loadtesting/shared/eks-vpc/terraform.tfstate"
    region         = "us-east-2"
    encrypt        = true
    kms_key_id     = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

provider "docker" {
  # Configuration options
  registry_auth {
    address  = "${data.aws_caller_identity.current.account_id}.dkr.ecr.us-east-2.amazonaws.com"
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

provider "git" {}

# Data sources for SigNoz EKS cluster authentication
data "aws_eks_cluster" "signoz" {
  count = var.enable_otel ? 1 : 0
  name  = module.signoz[0].cluster_name
}

data "aws_eks_cluster_auth" "signoz" {
  count = var.enable_otel ? 1 : 0
  name  = module.signoz[0].cluster_name
}

# Helm provider for SigNoz EKS cluster
provider "helm" {
  alias = "signoz"

  kubernetes {
    host                   = var.enable_otel ? data.aws_eks_cluster.signoz[0].endpoint : ""
    cluster_ca_certificate = var.enable_otel ? base64decode(data.aws_eks_cluster.signoz[0].certificate_authority[0].data) : ""
    token                  = var.enable_otel ? data.aws_eks_cluster_auth.signoz[0].token : ""
  }
}

# Kubernetes provider for SigNoz EKS cluster
provider "kubernetes" {
  alias = "signoz"

  host                   = var.enable_otel ? data.aws_eks_cluster.signoz[0].endpoint : ""
  cluster_ca_certificate = var.enable_otel ? base64decode(data.aws_eks_cluster.signoz[0].certificate_authority[0].data) : ""
  token                  = var.enable_otel ? data.aws_eks_cluster_auth.signoz[0].token : ""
}