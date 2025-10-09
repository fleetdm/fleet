terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.68.0"
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
}

data "aws_availability_zones" "available" {
  filter {
    name   = "region-name"
    values = [var.aws_region]
  }
}

locals {
  cluster_name = var.cluster_name
  azs          = slice(data.aws_availability_zones.available.names, 0, 2)
}

# Simplified VPC
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = local.cluster_name
  cidr = "10.0.0.0/16"

  azs             = local.azs
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }
}

# Simplified EKS
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 21.0"

  name               = local.cluster_name
  kubernetes_version = "1.31"

  endpoint_public_access = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # Managed node group
  eks_managed_node_groups = {
    default = {
      min_size       = 2
      max_size       = 2
      desired_size   = 2
      instance_types = ["t3.large"]

      # IAM policies for EBS CSI driver
      iam_role_additional_policies = {
        AmazonEBSCSIDriverPolicy = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
      }
    }
  }

  # Enable cluster creator admin access
  enable_cluster_creator_admin_permissions = true
}

# Explicit EKS Addons - install BEFORE node group completes
resource "aws_eks_addon" "vpc_cni" {
  cluster_name = module.eks.cluster_name
  addon_name   = "vpc-cni"

  depends_on = [module.eks]
}

resource "aws_eks_addon" "kube_proxy" {
  cluster_name = module.eks.cluster_name
  addon_name   = "kube-proxy"

  depends_on = [aws_eks_addon.vpc_cni]
}

resource "aws_eks_addon" "coredns" {
  cluster_name = module.eks.cluster_name
  addon_name   = "coredns"

  depends_on = [aws_eks_addon.kube_proxy]
}

resource "aws_eks_addon" "ebs_csi" {
  cluster_name = module.eks.cluster_name
  addon_name   = "aws-ebs-csi-driver"

  depends_on = [aws_eks_addon.coredns]
}

# Set gp2 as default storage class
resource "kubernetes_annotations" "gp2_default" {
  api_version = "storage.k8s.io/v1"
  kind        = "StorageClass"
  metadata {
    name = "gp2"
  }
  annotations = {
    "storageclass.kubernetes.io/is-default-class" = "true"
  }

  depends_on = [module.eks]
}

# SigNoz via Helm
resource "helm_release" "signoz" {
  name       = "signoz"
  repository = "https://charts.signoz.io"
  chart      = "signoz"
  namespace  = "signoz"
  timeout    = 900

  create_namespace = true

  set {
    name  = "cloud"
    value = "false"
  }

  set {
    name  = "signoz.service.type"
    value = "LoadBalancer"
  }

  # OTLP collector should be internal only (not publicly accessible)
  set {
    name  = "otelCollector.service.type"
    value = "LoadBalancer"
  }

  set {
    name  = "otelCollector.service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-scheme"
    value = "internal"
  }

  set {
    name  = "clickhouse.persistence.size"
    value = "20Gi"
  }

  depends_on = [
    aws_eks_addon.ebs_csi,
    module.eks
  ]
}
