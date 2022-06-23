provider "kubernetes" {
  experiments {
    manifest_resource = true
  }
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.cluster.token
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.cluster.endpoint
    token                  = data.aws_eks_cluster_auth.cluster.token
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  }
}

provider "kubectl" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.cluster.token
  load_config_file       = false
  apply_retry_count      = 5
}

locals {
  cluster_version = "1.21"
}

output "eks_cluster" {
  value = module.aws-eks-accelerator-for-terraform
}

terraform {
  required_providers {
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "1.14.0"
    }
  }
}

data "aws_iam_role" "admin" {
  name = "admin"
}

module "aws-eks-accelerator-for-terraform" {
  source       = "github.com/aws-samples/aws-eks-accelerator-for-terraform.git"
  cluster_name = var.prefix

  # EKS Cluster VPC and Subnets
  vpc_id             = var.vpc.vpc_id
  private_subnet_ids = var.vpc.private_subnets

  # EKS CONTROL PLANE VARIABLES
  cluster_version = local.cluster_version

  # EKS MANAGED NODE GROUPS
  managed_node_groups = {
    mg_4 = {
      node_group_name = "managed-ondemand"
      instance_types  = ["t3.medium"]
      subnet_ids      = var.vpc.private_subnets
    }
  }

  map_roles = concat([for i in var.eks_allowed_roles : {
    rolearn  = i.arn
    username = i.id
    groups   = ["system:masters"]
    }], [{
    rolearn  = data.aws_iam_role.admin.arn
    username = data.aws_iam_role.admin.id
    groups   = ["system:masters"]
  }])
}

data "aws_eks_cluster" "cluster" {
  name = module.aws-eks-accelerator-for-terraform.eks_cluster_id
}

data "aws_eks_cluster_auth" "cluster" {
  name = module.aws-eks-accelerator-for-terraform.eks_cluster_id
}

module "kubernetes-addons" {
  source = "github.com/aws-samples/aws-eks-accelerator-for-terraform.git//modules/kubernetes-addons"

  eks_cluster_id               = module.aws-eks-accelerator-for-terraform.eks_cluster_id
  eks_cluster_endpoint         = module.aws-eks-accelerator-for-terraform.eks_cluster_endpoint
  eks_cluster_version          = local.cluster_version
  eks_oidc_provider            = module.aws-eks-accelerator-for-terraform.eks_oidc_issuer_url
  eks_worker_security_group_id = module.aws-eks-accelerator-for-terraform.worker_node_security_group_id

  # EKS Managed Add-ons
  enable_amazon_eks_vpc_cni            = true
  enable_amazon_eks_coredns            = true
  enable_amazon_eks_kube_proxy         = true
  enable_amazon_eks_aws_ebs_csi_driver = true

  #K8s Add-ons
  enable_aws_load_balancer_controller = false
  enable_metrics_server               = false
  enable_cluster_autoscaler           = true
  enable_vpa                          = true
  enable_prometheus                   = false
  enable_ingress_nginx                = true
  enable_aws_for_fluentbit            = false
  enable_argocd                       = false
  enable_fargate_fluentbit            = false
  enable_argo_rollouts                = false
  enable_kubernetes_dashboard         = false
  enable_yunikorn                     = false

  depends_on = [module.aws-eks-accelerator-for-terraform.managed_node_groups]
}
