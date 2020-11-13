# Monitoring Fleet

## Health Checks

Fleet exposes a basic health check at the `/healthz` endpoint. This is the interface to use for simple monitoring and load-balancer health checks.

The `/healthz` endpoint will return an `HTTP 200` status if the server is running and has healthy connections to MySQL and Redis. If there are any problems, the endpoint will return an `HTTP 500` status.

## Metrics

Fleet exposes server metrics in a format compatible with [Prometheus](https://prometheus.io/). A simple example Prometheus configuration is available in [tools/app/prometheus.yml](/tools/app/prometheus.yml).

Prometheus can be configured to use a wide range of service discovery mechanisms within AWS, GCP, Azure, Kubernetes, and more. See the Prometheus [configuration documentation](https://prometheus.io/docs/prometheus/latest/configuration/configuration/) for more information on configuring the

### Alerting

Prometheus has built-in support for alerting through [Alertmanager](https://prometheus.io/docs/alerting/latest/overview/).

Consider building alerts for

- Changes from expected levels of host enrollment
- Increased latency on HTTP endpoints
- Increased error levels on HTTP endpoints

```
TODO (Seeking Contributors)
Add example alerting configurations
```

### Graphing

Prometheus provides basic graphing capabilities, and integrates tightly with [Grafana](https://prometheus.io/docs/visualization/grafana/) for sophisticated visualizations.
