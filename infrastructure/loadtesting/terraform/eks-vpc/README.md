# EKS VPC for Fleet Loadtesting

Dedicated VPC for EKS workloads (SigNoz) with proper Kubernetes tags.

## Architecture

- **CIDR**: 10.20.0.0/16
- **Subnets**: 2 AZs (us-east-2a, us-east-2b)
  - Private: 10.20.1.0/24, 10.20.2.0/24
  - Public: 10.20.101.0/24, 10.20.102.0/24
- **NAT**: Single NAT gateway (cost optimization)
- **Tags**: Pre-configured for EKS/Kubernetes

## Usage

This VPC is deployed per workspace:

```bash
cd infrastructure/loadtesting/terraform/eks-vpc
terraform workspace new <workspace_name>
terraform apply
```

The VPC outputs are consumed by the SigNoz module via terraform remote state.

## Why Separate VPC?

- EKS requires specific subnet tags (`kubernetes.io/cluster/*`)
- Can't modify shared fleet-vpc tags (different terraform state)
- Avoids VPC limit issues (dedicated EKS VPC)
- Clean separation of concerns
