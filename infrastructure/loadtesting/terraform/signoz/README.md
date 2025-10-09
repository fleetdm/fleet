# SigNoz module for Fleet loadtest OTEL tracing

OpenTelemetry observability backend module, deployed conditionally from the Fleet loadtest infrastructure.

## Usage

This module is deployed automatically when `enable_otel=true` is set in the parent loadtest infrastructure:

```bash
cd infrastructure/loadtesting/terraform/infra
terraform workspace new <workspace_name>
terraform apply -var=enable_otel=true
```

## What gets deployed

- **Separate EKS cluster** for SigNoz (K8s 1.31, 2x t3.large nodes)
- **OTLP endpoint**: Internal LoadBalancer (not publicly accessible)
- **SigNoz UI**: Public LoadBalancer on port 8080
- **Storage**: EBS CSI driver with gp2 default storage class
