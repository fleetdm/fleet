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
      configuration_aliases = [helm]
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
      configuration_aliases = [kubernetes]
    }
  }
}

locals {
  cluster_name = var.cluster_name
}

# Use shared fleet VPC
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 21.0"

  name               = local.cluster_name
  kubernetes_version = "1.31"

  endpoint_public_access = true

  vpc_id     = var.vpc_id
  subnet_ids = var.subnet_ids

  # IMPORTANT: Install critical addons BEFORE node group to avoid circular dependency
  # Nodes need VPC CNI to become Ready, but terraform waits for nodes to be Ready
  # before creating addons. This causes a deadlock where nodes are stuck NotReady.
  # Solution: Use addons with before_compute=true to install VPC CNI before node group completes.
  addons = {
    vpc-cni = {
      most_recent    = true
      before_compute = true
    }
    kube-proxy = {
      most_recent    = true
      before_compute = true
    }
    coredns = {
      most_recent    = true
      before_compute = true
    }
  }

  # Managed node group
  eks_managed_node_groups = {
    default = {
      min_size       = 2
      max_size       = 2
      desired_size   = 2
      instance_types = ["t3.large"]
    }
  }

  # Enable cluster creator admin access
  enable_cluster_creator_admin_permissions = true

  # Enable OIDC provider for IRSA (IAM Roles for Service Accounts)
  enable_irsa = true
}

# IAM Role for EBS CSI Driver Service Account (IRSA)
module "ebs_csi_irsa_role" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.0"

  role_name             = "${local.cluster_name}-ebs-csi-driver"
  attach_ebs_csi_policy = true

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
    }
  }
}

# EBS CSI Driver addon with IRSA support
# This must be created separately to avoid circular dependency with OIDC provider
resource "aws_eks_addon" "ebs_csi" {
  cluster_name             = module.eks.cluster_name
  addon_name               = "aws-ebs-csi-driver"
  addon_version            = data.aws_eks_addon_version.ebs_csi.version
  service_account_role_arn = module.ebs_csi_irsa_role.iam_role_arn

  depends_on = [
    module.ebs_csi_irsa_role,
    module.eks
  ]
}

data "aws_eks_addon_version" "ebs_csi" {
  addon_name         = "aws-ebs-csi-driver"
  kubernetes_version = module.eks.cluster_version
  most_recent        = true
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

  set {
    name  = "clickhouse.persistence.storageClassName"
    value = "gp3"
  }

  depends_on = [
    module.eks
  ]
}
