Working With Osquery Logs
=========================

The `kolide` binary accepts two flags:

- `--osquery_result_log_file`: Path for osqueryd result logs (default: `/tmp/osquery_result`)
- `--osquery_status_log_file`: Path for osqueryd status logs (default `/tmp/osquery_status`)

You can also configure the path which logs are written via environment variables or a config file. See the documentation on [Configuring The Kolide Binary](../infrastructure/configuring-the-kolide-binary.md) for more information on this.

As the Kolide server ingests logs from osquery, it will write them to the paths described using the above flags. You are encouraged to forward these logs into your company's log aggregation/alerting pipeline directly. For more information on configuring various systems to ingest osquery logs, consider reviewing the [Log Aggregation](https://osquery.readthedocs.io/en/stable/deployment/log-aggregation/) documentation on the official osquery wiki.

As the Kolide application grows, we are going to expand this feature based on customer feedback. If you would like direct integrations with a specific third-party application, please let us know at [support@kolide.co](mailto:support@kolide.co).
