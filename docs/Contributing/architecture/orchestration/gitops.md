# GitOps

## Summary

GitOps is a way to manage the Fleet server using Git as the source of truth for declarative infrastructure and deployment configurations. This approach ensures consistency, auditability, and enables scalable management across environments.

## Goals

- Define infrastructure and application state declaratively in Git.
- Use automated controllers to reconcile desired state with cluster state.
- Reduce manual operations and prevent configuration drift.
- Provide traceability and auditability via Git history.

## Non-goals

- Handling imperative or ad-hoc changes outside the Git workflow.
- Replacing CI pipelines for testing or building artifacts.
- Managing stateful data or secrets without external tooling.

## How it works

The GitOps workflow in Fleet relies on a set of components:

- A Git repository contains the source of truth, including Kubernetes manifests, Helm charts, or Kustomize overlays.
- The Fleet controller (running in a management cluster) watches the repository for changes.
- When a change is detected, the controller creates a bundle representing the desired state.
- Agents running in managed clusters pull the bundle and apply the manifests to the local cluster.
- The agent reports deployment and health status back to the controller.

This pull-based architecture supports secure, scalable multi-cluster deployments.

## Data flow

1. A user commits Kubernetes configurations to a Git repository.
2. Fleet detects a change in the repository (via polling or webhook).
3. Fleet creates or updates bundles representing the new desired state.
4. The agent running in each cluster pulls the appropriate bundle.
5. The agent applies the manifests using `kubectl apply` semantics.
6. The agent reports success or failure to the controller.

## Security

- Fleet agents pull configurations, so no inbound connections to clusters are required.
- Each cluster has scoped credentials and service accounts.
- Git credentials (SSH, HTTPS) are stored as Kubernetes secrets and mounted into the controller pods.
- Registration tokens are short-lived and scoped per-cluster.

## Scalability

Fleet is designed to scale to thousands of clusters. It uses bundle deduplication and streaming updates to reduce load and improve performance. Agents operate independently, allowing parallel deployments and fault isolation.

## Failure modes

- If the Git repository is unavailable, no updates will be deployed, but existing configurations are unaffected.
- If an agent cannot connect to the controller, it will retry periodically.
- Misconfigured manifests can lead to partial or failed deployments; these are reported back for remediation.

## Future work

- Improved observability for bundle status and diffs.
- Support for validation and policy enforcement on commits.
- Integration with secrets management tools (e.g., Vault, Sealed Secrets).

