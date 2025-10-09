# SigNoz Simple Deployment

Deployment of SigNoz OpenTelemetry backend on EKS for loadtesting.

## Key Features

- Uses EKS module v20 (stable)
- Explicit `aws_eks_addon` resources for vpc-cni, kube-proxy, coredns, aws-ebs-csi-driver
- IAM permissions for EBS CSI driver on node group
- Default gp2 storage class configured
- Public LoadBalancer endpoints for SigNoz UI and OTLP collector
- K8s 1.31, 2x t3.large nodes

## Prerequisites

AWS SSO configured with loadtesting profile:
```bash
aws sso login
```

## Quick Start

```bash
cd infrastructure/loadtesting/terraform/signoz
terraform init
terraform workspace new signoz-simple  # or select existing
terraform apply
```

Deployment takes ~10-15 minutes (EBS CSI addon takes longest).

## Getting Endpoints

After deployment completes:

```bash
# Configure kubectl to connect to the EKS cluster
$(terraform output -raw configure_kubectl)

# Wait for LoadBalancer provisioning (2-3 minutes)
kubectl get svc -n signoz

# Get SigNoz UI URL (returns: http://hostname:8080)
echo "http://$(terraform output -raw get_signoz_ui_url)"

# Get OTLP endpoint for your app (returns: hostname:4317)
terraform output -raw get_otlp_endpoint
```

## Using with Local Apps

Point your local application's OpenTelemetry configuration to:
```
OTLP_ENDPOINT=$(terraform output -raw get_otlp_endpoint)
```

## Troubleshooting

Check pod status:
```bash
kubectl get pods -n signoz
```

Check EBS CSI driver (must be Running 6/6):
```bash
kubectl get pods -n kube-system | grep ebs-csi-controller
```

Check PVC status (must be Bound):
```bash
kubectl get pvc -n signoz
```

## Cleanup

```bash
terraform destroy
```
