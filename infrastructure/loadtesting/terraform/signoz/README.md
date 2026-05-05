# SigNoz for Fleet Loadtesting

SigNoz provides OpenTelemetry tracing for Fleet loadtest environments. It's deployed as a standalone Terraform root module to ensure it's available before Fleet starts up.

## Architecture

- **EKS Cluster**: Per-workspace (e.g., `signoz-victor-baseline`)
- **Kubernetes**: v1.31
- **Node group**: 2x t3.xlarge nodes
- **Components**:
  - SigNoz UI (public LoadBalancer on port 8080)
  - OTLP Collector (internal LoadBalancer on port 4317)
  - ClickHouse (200Gi storage)

## Deployment order

**IMPORTANT**: SigNoz must be deployed BEFORE the main Fleet infrastructure to capture telemetry from Fleet's initial bootup.

1. Deploy shared EKS VPC (one time, shared across workspaces, should already be deployed)
2. Deploy SigNoz (this directory)
3. Deploy Fleet infrastructure (../infra)

## Usage

```bash
# 1. Initialize and select workspace
cd infrastructure/loadtesting/terraform/signoz
terraform init
terraform workspace new <workspace_name>  # Match your infra workspace

# 2. Deploy SigNoz
terraform apply

# 3. Wait for deployment to complete (~10-15 minutes)
# The OTLP collector endpoint will be shown in outputs

# 4. Now deploy Fleet infrastructure
cd ../infra
terraform apply
```

## Accessing SigNoz UI

```bash
# Get the SigNoz UI URL
terraform output -raw get_signoz_ui_url | bash

# Or configure kubectl and access directly
$(terraform output -raw configure_kubectl)
kubectl get svc -n signoz signoz -o jsonpath='http://{.status.loadBalancer.ingress[0].hostname}:8080'
```

## Managing storage and retention

**IMPORTANT**: ClickHouse has limited storage. To prevent running out of space:

1. **Reduce trace retention period** in the SigNoz UI:
    - Navigate to Settings → Retention Period
    - Lower the retention period for traces (default may be too long for loadtesting)
    - Consider 1-3 days for active loadtest environments

2. **Monitor ClickHouse storage**:
   ```bash
   # Check ClickHouse pod storage usage
   kubectl exec -n signoz chi-signoz-clickhouse-cluster-0-0-0 -- df -h /var/lib/clickhouse

   # Check database sizes
   kubectl exec -n signoz chi-signoz-clickhouse-cluster-0-0-0 -- clickhouse-client --query "SELECT database, formatReadableSize(sum(bytes_on_disk)) AS size FROM system.parts WHERE active GROUP BY database ORDER BY sum(bytes_on_disk) DESC"
   ```

3. **What happens when storage is full**:
    - ClickHouse will reject new writes
    - **New traces will NOT be captured**
    - OTEL collector will log errors about failed writes
    - Fleet will continue running but traces will be lost

## Outputs

The main Fleet infrastructure reads these outputs via remote state:
- `cluster_name`: EKS cluster name
- `otel_collector_endpoint`: Internal OTLP endpoint for Fleet to send traces
- `configure_kubectl`: Command to configure kubectl access

## Destroying

```bash
terraform destroy
```
