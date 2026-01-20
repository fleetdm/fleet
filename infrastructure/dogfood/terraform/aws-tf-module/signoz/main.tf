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

  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "dogfood/signoz/terraform.tfstate"
    workspace_key_prefix = "dogfood"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-dogfood"
    }
  }
}

provider "aws" {
  default_tags {
    tags = {
      environment = "dogfood-signoz"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws-tf-module/signoz"
      workspace   = terraform.workspace
    }
  }
}

# Read shared VPC from remote state
data "terraform_remote_state" "dogfood" {
  backend   = "s3"
  workspace = "fleet"
  config = {
    bucket               = "fleet-terraform-remote-state"
    key                  = "fleet"
    region               = "us-east-2"
    dynamodb_table       = "fleet-terraform-state-lock"
  }
}

# Providers for Helm/Kubernetes after cluster is created
provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    token                  = data.aws_eks_cluster_auth.signoz.token
  }
}

provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
  token                  = data.aws_eks_cluster_auth.signoz.token
}

data "aws_region" "current" {}

data "aws_eks_cluster_auth" "signoz" {
  name = module.eks.cluster_name
}

data "aws_route53_zone" "dogfood" {
  name = "dogfood.fleetdm.com."
}

locals {
  cluster_name    = "signoz"
  signoz_domain   = "signoz.dogfood.fleetdm.com"
  signoz_alb_name = "signoz-dogfood"
}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = local.signoz_domain
  zone_id     = data.aws_route53_zone.dogfood.zone_id

  wait_for_validation = true
}

# Use shared fleet VPC
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 21.0"

  name               = local.cluster_name
  kubernetes_version = "1.31"

  endpoint_public_access = true

  vpc_id     = data.terraform_remote_state.dogfood.outputs.vpc.vpc_id
  subnet_ids = data.terraform_remote_state.dogfood.outputs.vpc.private_subnets

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
      min_size       = 1
      max_size       = 1
      desired_size   = 1
      instance_types = ["t3.2xlarge"]
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

# Wait for EBS CSI driver to be active
resource "time_sleep" "wait_for_ebs_csi" {
  depends_on = [aws_eks_addon.ebs_csi]

  create_duration = "60s"
}

# Create gp3 storage class using EBS CSI driver
resource "kubernetes_storage_class_v1" "gp3" {
  metadata {
    name = "gp3"
    annotations = {
      "storageclass.kubernetes.io/is-default-class" = "true"
    }
  }

  storage_provisioner    = "ebs.csi.aws.com"
  reclaim_policy         = "Delete"
  allow_volume_expansion = true
  volume_binding_mode    = "WaitForFirstConsumer"

  parameters = {
    type      = "gp3"
    encrypted = "true"
    fsType    = "ext4"
  }

  depends_on = [time_sleep.wait_for_ebs_csi]
}

# SigNoz via Helm
resource "helm_release" "signoz" {
  name       = "signoz"
  repository = "https://charts.signoz.io"
  chart      = "signoz"
  namespace  = "signoz"
  timeout    = 3600 # 60 minutes - allows for worst-case AWS delays during both apply and destroy

  create_namespace = true
  wait             = true
  wait_for_jobs    = false

  # OTEL Collector configuration overrides for production stability
  values = [
    file("${path.module}/otel-collector-values.yaml")
  ]

  set {
    name  = "cloud"
    value = "false"
  }

  set {
    name  = "signoz.service.type"
    value = "ClusterIP"
  }

  set {
    name  = "signoz.ingress.enabled"
    value = "true"
  }

  set {
    name  = "signoz.ingress.className"
    value = "alb"
  }

  set {
    name  = "signoz.ingress.annotations.alb\\.ingress\\.kubernetes\\.io/scheme"
    value = "internal"
  }

  set {
    name  = "signoz.ingress.annotations.alb\\.ingress\\.kubernetes\\.io/listen-ports"
    value = "[{\"HTTPS\":443}]"
  }

  set {
    name  = "signoz.ingress.annotations.alb\\.ingress\\.kubernetes\\.io/certificate-arn"
    value = module.acm.acm_certificate_arn
  }

  set {
    name  = "signoz.ingress.annotations.alb\\.ingress\\.kubernetes\\.io/backend-protocol"
    value = "HTTP"
  }

  set {
    name  = "signoz.ingress.annotations.alb\\.ingress\\.kubernetes\\.io/target-type"
    value = "ip"
  }

  set {
    name  = "signoz.ingress.annotations.alb\\.ingress\\.kubernetes\\.io/load-balancer-name"
    value = local.signoz_alb_name
  }

  set {
    name  = "signoz.ingress.hosts[0].host"
    value = local.signoz_domain
  }

  set {
    name  = "signoz.ingress.hosts[0].paths[0].path"
    value = "/"
  }

  set {
    name  = "signoz.ingress.hosts[0].paths[0].pathType"
    value = "ImplementationSpecific"
  }

  set {
    name  = "signoz.ingress.hosts[0].paths[0].port"
    value = "8080"
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


  # Clickhouse storage configuration
  set {
    name  = "clickhouse.persistence.size"
    value = "200Gi"
  }

  set {
    name  = "clickhouse.persistence.storageClass"
    value = "gp3"
  }

  # Zookeeper storage configuration
  set {
    name  = "zookeeper.persistence.storageClass"
    value = "gp3"
  }

  # SigNoz (alertmanager) storage configuration
  set {
    name  = "alertmanager.persistence.storageClass"
    value = "gp3"
  }

  # ClickHouse resource configuration for loadtest
  # Default 100m CPU and 200Mi memory are way too low for high-volume telemetry
  set {
    name  = "clickhouse.resources.requests.cpu"
    value = "2000m"
  }

  set {
    name  = "clickhouse.resources.requests.memory"
    value = "4Gi"
  }

  set {
    name  = "clickhouse.resources.limits.cpu"
    value = "4000m"
  }

  set {
    name  = "clickhouse.resources.limits.memory"
    value = "8Gi"
  }

  # OTEL Collector resource configuration for loadtest
  set {
    name  = "otelCollector.resources.requests.memory"
    value = "8Gi"
  }

  set {
    name  = "otelCollector.resources.limits.memory"
    value = "12Gi"
  }

  set {
    name  = "otelCollector.resources.requests.cpu"
    value = "1000m"
  }

  set {
    name  = "otelCollector.resources.limits.cpu"
    value = "4000m"
  }

  # Only need 1 replica since we have 1 LoadBalancer endpoint
  set {
    name  = "otelCollector.replicaCount"
    value = "1"
  }

  depends_on = [
    module.eks,
    kubernetes_storage_class_v1.gp3
  ]
}

data "aws_lb" "signoz" {
  name       = local.signoz_alb_name
  depends_on = [helm_release.signoz]
}

resource "aws_route53_record" "signoz" {
  zone_id = data.aws_route53_zone.dogfood.zone_id
  name    = local.signoz_domain
  type    = "A"

  alias {
    name                   = data.aws_lb.signoz.dns_name
    zone_id                = data.aws_lb.signoz.zone_id
    evaluate_target_health = true
  }
}
