# Osquery Logs
- [Osquery logging plugins](#osquery-logging-plugins)
  - [Filesystem](#filesystem)
  - [Firehose](#firehose)
  - [Kinesis](#kinesis)
  - [Lambda](#lambda)
  - [PubSub](#pubsub)
  - [Stdout](#stdout)
- [Advanced Destinations](#advanced-destinations)
  - [Snowflake via AWS Firehose](#snowflake)
  - [Splunk via AWS Firehose](#splunk)

Osquery agents are typically configured to send logs to the Fleet server (`--logger_plugin=tls`). This is not a requirement, and any other logger plugin can be used even when osquery clients are connecting to the Fleet server to retrieve configuration or run live queries. See the [osquery logging documentation](https://osquery.readthedocs.io/en/stable/deployment/logging/) for more about configuring logging on the agent.

If `--logger_plugin=tls` is used with osquery clients, the following configuration can be applied on the Fleet server for handling the incoming logs.

## Osquery logging plugins

Fleet supports the following logging plugins for osquery logs:

- [Filesystem](#filesystem) - Logs are written to the local Fleet server filesystem.
- [Firehose](#firehose) - Logs are written to AWS Firehose streams.
- [Kinesis](#kinesis) - Logs are written to AWS Kinesis streams.
- [Lambda](#lambda) - Logs are written to AWS Lambda functions.
- [PubSub](#pubsub) - Logs are written to Google Cloud PubSub topics.
- [Stdout](#stdout) - Logs are written to stdout.

To set the osquery logging plugins, use the `--osquery_result_log_plugin` and `--osquery_status_log_plugin` flags (or [equivalents for environment variables or configuration files](../02-Deploying/02-Configuration.md#options)).

### Filesystem

The default logging plugin.

- Plugin name: `filesystem`
- Flag namespace: [filesystem](../02-Deploying/02-Configuration.md#filesystem)

With the filesystem plugin, osquery result and/or status logs are written to the local filesystem on the Fleet server. This is typically used with a log forwarding agent on the Fleet server that will push the logs into a logging pipeline. Note that if multiple load-balanced Fleet servers are used, the logs will be load-balanced across those servers (not duplicated).

### Firehose

- Plugin name: `firehose`
- Flag namespace: [firehose](../02-Deploying/02-Configuration.md#firehose)

With the Firehose plugin, osquery result and/or status logs are written to [AWS Firehose](https://aws.amazon.com/kinesis/data-firehose/) streams. This is a very good method for aggregating osquery logs into AWS S3 storage.

Note that Firehose logging has limits [discussed in the documentation](https://docs.aws.amazon.com/firehose/latest/dev/limits.html). When Fleet encounters logs that are too big for Firehose, notifications will be output in the Fleet logs and those logs _will not_ be sent to Firehose.

### Kinesis

- Plugin name: `kinesis`
- Flag namespace: [kinesis](../02-Deploying/02-Configuration.md#kinesis)

With the Kinesis plugin, osquery result and/or status logs are written to
[AWS Kinesis](https://aws.amazon.com/kinesis/data-streams) streams.

Note that Kinesis logging has limits [discussed in the
documentation](https://docs.aws.amazon.com/kinesis/latest/dev/limits.html).
When Fleet encounters logs that are too big for Kinesis, notifications will be
output in the Fleet logs and those logs _will not_ be sent to Kinesis.

### Lambda

- Plugin name: `lambda`
- Flag namespace: [lambda](../02-Deploying/02-Configuration.md#lambda)

With the Lambda plugin, osquery result and/or status logs are written to
[AWS Lambda](https://aws.amazon.com/lambda/) functions.

Lambda processes logs from Fleet synchronously, so the Lambda function used must not take enough processing time that the osquery client times out while writing logs. If there is heavy processing to be done, use Lambda to store the logs in another datastore/queue before performing the long-running process.

Note that Lambda logging has limits [discussed in the
documentation](https://docs.aws.amazon.com/lambda/latest/dg/gettingstarted-limits.html). The maximum size of a log sent to Lambda is 6MB.
When Fleet encounters logs that are too big for Lambda, notifications will be
output in the Fleet logs and those logs _will not_ be sent to Lambda.

Lambda is executed once per log line. As a result, queries with `differential` result logging might result in a higher number of Lambda invocations.

> Queries are assigned `differential` result logging by default in Fleet. `differential` logs have two format options, single (event) and batched. [Check out the osquery documentation](https://osquery.readthedocs.io/en/stable/deployment/logging/#differential-logs) for more information on `differential` logs.

Keep this in mind when using Lambda, as you're charged based on the number of requests for your functions and the duration, the time it takes for your code to execute. 

### PubSub

- Plugin name: `pubsub`
- Flag namespace: [pubsub](../02-Deploying/02-Configuration.md#pubsub)

With the PubSub plugin, osquery result and/or status logs are written to [PubSub](https://cloud.google.com/pubsub/) topics.

Note that messages over 10MB will be dropped, with a notification sent to the fleet logs, as these can never be processed by PubSub.

### Stdout

- Plugin name: `stdout`
- Flag namespace: [stdout](../02-Deploying/02-Configuration.md#stdout)

With the stdout plugin, osquery result and/or status logs are written to stdout
on the Fleet server. This is typically used for debugging or with a log
forwarding setup that will capture and forward stdout logs into a logging
pipeline. Note that if multiple load-balanced Fleet servers are used, the logs
will be load-balanced across those servers (not duplicated).

## Advanced Destinations

### Snowflake

After configuring Fleet to use [firehose logging plugin](#firehose) you can enable Firehose to Snowflake with the
Snowpipe integration. Using [firehose.tf](https://github.com/fleetdm/fleet/blob/main/tools/terraform/firehose.tf)
and the [AWS guide](https://docs.aws.amazon.com/prescriptive-guidance/latest/patterns/automate-data-stream-ingestion-into-a-snowflake-database-by-using-snowflake-snowpipe-amazon-s3-amazon-sns-and-amazon-kinesis-data-firehose.html)
its possible to continuously ship osquery results to Snowflake. Snowflake also provides some [details](https://docs.snowflake.com/en/user-guide/data-load-snowpipe-auto-s3.html#prerequisite-create-an-amazon-sns-topic-and-subscription)
to get the destination tables and IAM roles required in AWS prepared.

### Splunk

After configuring Fleet to use [firehose logging plugin](#firehose) you can enable Firehose to forward directly to Splunk
as a [supported destination](https://docs.aws.amazon.com/firehose/latest/dev/create-destination.html#create-destination-splunk).
Using [firehose.tf](https://github.com/fleetdm/fleet/blob/main/tools/terraform/firehose.tf) as a starting point replace the S3
destination with [a splunk destination](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream#splunk-destination)
. Splunk also provides [a guide](https://docs.splunk.com/Documentation/AddOns/latest/Firehose/ConfigureFirehose) to prepare the
Splunk platform for the data.