# ADR-0005: Standardize on OpenTelemetry for observability 📊

## Status 🚦

Approved

## Date 📅

2025-08-13

## Context 🔍

Fleet currently supports multiple telemetry and observability formats:

- **Prometheus** for metrics (documented in public-facing server configuration) 📈
- **Sentry** for error tracking (documented in public-facing server configuration) 🚨
- **OpenTelemetry** for traces and metrics (documented internally in [/tools/telemetry/](https://github.com/fleetdm/fleet/blob/167e2e3e28d694e61f313afdb5dd92f615f2aa7b/tools/telemetry/README.md#L17)) 🔭
- **Elastic APM** for application performance monitoring (documented internally in [/tools/apm-elastic/](https://github.com/fleetdm/fleet/blob/7380919dc3bb817387d1b5f9f621b272d3a0d53e/tools/apm-elastic/README.md#L1-L0)) 📉

This fragmentation creates several challenges:

- 🔧 Increased maintenance burden supporting multiple telemetry integrations
- 😕 Inconsistent observability experiences across different deployment environments
- 🔄 Duplicated effort implementing similar functionality for different formats
- 🚪 Higher barrier to entry for users configuring observability
- 📚 Complexity in documentation and support

Additionally, there is a business need to use telemetry in both Fleet's cloud-hosted environments and customer on-premises deployments. Having a standardized observability experience across these environments would significantly improve our ability to support on-prem customers, as support teams could use familiar tools and workflows regardless of the deployment model.

OpenTelemetry has emerged as the industry standard for observability data collection and transmission, with widespread adoption across the ecosystem. It provides a vendor-neutral, standardized approach to instrumentation that supports metrics, traces, and logs in a unified framework.

## Decision 💡

Fleet will standardize on OpenTelemetry as the primary telemetry format for all observability data (metrics, traces, errors, etc.).

We will:

1. **Maintain OpenTelemetry** as the recommended observability solution 🎯
2. **Continue supporting existing formats** (Prometheus, Sentry) during a transition period ⏳
3. **Provide clear migration paths** for users currently using proprietary formats 🗺️
4. **Document OpenTelemetry configuration** in the public-facing documentation 📖
5. **Eventually deprecate** proprietary telemetry formats in favor of OpenTelemetry 🔄

Migration paths will be provided:

### Prometheus to OpenTelemetry migration 📈➡️🔭
- Install OpenTelemetry Collector (provide docker-compose example) 🐳
- Configure Fleet to send metrics to the Collector ⚙️
- Point Prometheus to scrape the Collector's `/metrics` endpoint 🎯

### Sentry to OpenTelemetry migration 🚨➡️🔭
- Leverage Sentry's direct ingestion of OTel data (https://docs.sentry.io/concepts/otlp/) 🔗
- Since Fleet only exports errors for Sentry, this approach matches the current use case ✅
- Configure Fleet to send error data via OpenTelemetry protocol to Sentry 📤

## Consequences 🎯

### Positive ✅
- **Unified observability**: Single protocol for metrics, traces, and errors 🎯
- **Industry standard**: Aligns with broader industry adoption and tooling 🌍
- **Vendor neutrality**: Users can easily switch between observability backends 🔄
- **Reduced maintenance**: Single telemetry integration to maintain 🛠️
- **Better ecosystem support**: Wide range of compatible tools and services 🤝
- **Future-proof**: OpenTelemetry is actively developed with strong community support 🚀

### Negative ⚠️
- **Migration effort**: Existing users need to update their configurations 🔧
- **Learning curve**: Users unfamiliar with OpenTelemetry need to learn new concepts 📚
- **Temporary complexity**: Supporting both old and new formats during transition 🔀
- **Feature parity**: Some proprietary features may not have direct OTel equivalents 🤷
- **Documentation updates**: Some documentation rewrite required 📝

## Alternatives considered 🤔

### Status quo: Continue supporting multiple formats
- **Pros**: ✅ No migration required, users can continue with familiar tools, no breaking changes
- **Cons**: ❌ Ongoing maintenance burden, inconsistent experiences, duplicated effort, higher complexity
- **Rejected because**: Long-term maintenance costs outweigh short-term migration effort

### Support only proprietary formats
- **Pros**: ✅ Simpler integrations with specific vendors, potentially richer feature sets
- **Cons**: ❌ Vendor lock-in, limited flexibility, higher cost for users, fragmented ecosystem
- **Rejected because**: Goes against Fleet's philosophy of openness and flexibility

## References 📚

- 📖 OpenTelemetry specification: https://opentelemetry.io/docs/specs/otel/
- 🐳 OpenTelemetry Collector documentation: https://opentelemetry.io/docs/collector/
- 📈 Fleet's Prometheus configuration: https://fleetdm.com/docs/configuration/fleet-server-configuration#prometheus
- 🔔 Fleet's Sentry configuration: https://fleetdm.com/docs/configuration/fleet-server-configuration#sentry
- 🔧 Related engineering initiated issue that will identify implementation steps: https://github.com/fleetdm/fleet/issues/30253
- 📐 [Telemetry attribute naming guide](../guides/telemetry-attribute-naming.md) - conventions for naming log, trace, and metric attributes
- 📐 [ADR-0009: Attribute naming conventions](0009-attribute-naming-conventions.md) - decision record for attribute naming
