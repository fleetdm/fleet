# Osquery logs

This document provides instructions for working with each of the following log destinations in Fleet.

To configure each log destination, you must set the correct osquery logging configuration options in Fleet. Check out the reference documentation for osquery logging configuration options [here in the Fleet documentation](../02-Deploying/03-Configuration.md#osquery-status-log-plugin).

- [Firehose](#firehose)
- [Snowflake](#snowflake)
- [Splunk](#splunk)
- [Kinesis](#kinesis)
- [Lambda](#lambda)
- [PubSub](#pubsub)
- [Stdout](#stdout)
- [Filesystem](#filesystem)

### Firehose

Logs are written to AWS Firehose streams.

- Plugin name: `firehose`
- Flag namespace: [firehose](../02-Deploying/03-Configuration.md#firehose)

With the Firehose plugin, osquery result and/or status logs are written to [Amazon Kinesis Data Firehose](https://aws.amazon.com/kinesis/data-firehose/). This is a very good method for aggregating osquery logs into AWS S3 storage.

Note that Firehose logging has limits [discussed in the documentation](https://docs.aws.amazon.com/firehose/latest/dev/limits.html). When Fleet encounters logs that are too big for Firehose, notifications will be output in the Fleet logs and those logs _will not_ be sent to Firehose.

### Snowflake

To send logs to Snowflake, you must first configure Fleet to send logs to [Firehose](#firehose). This is because you'll use the Snowflake Snowpipe integration to direct logs to Snowflake.

If you're using Fleet's [terraform reference architecture](https://github.com/fleetdm/fleet/blob/main/tools/terraform/firehose.tf), Firehose is already configured as your log destination.

With Fleet configured to send logs to Firehose, you then want to load the data from Firehose into a Snowflake database. AWS provides instructions on how to direct logs to a Snowflake database [here in the AWS documentation](https://docs.aws.amazon.com/prescriptive-guidance/latest/patterns/automate-data-stream-ingestion-into-a-snowflake-database-by-using-snowflake-snowpipe-amazon-s3-amazon-sns-and-amazon-kinesis-data-firehose.html)

Snowflake provides instructions on setting up the destination tables and IAM roles required in AWS [here in the Snowflake docs](https://docs.snowflake.com/en/user-guide/data-load-snowpipe-auto-s3.html#prerequisite-create-an-amazon-sns-topic-and-subscription).

### Splunk

To send logs to Splunk, you must first configure Fleet to send logs to [Firehose](#firehose). This is because you'll enable Firehose to forward logs directly to Splunk.

With Fleet configured to send logs to Firehose, you then want to load the data from Firehose into Splunk. AWS provides instructions on how to enable Firehose to forward directly to Splunk [here in the AWS documentation](https://docs.aws.amazon.com/firehose/latest/dev/create-destination.html#create-destination-splunk).

If you're using Fleet's [terraform reference architecture](https://github.com/fleetdm/fleet/blob/main/tools/terraform), you want to replace the S3 destination with a Splunk destination. Hashicorp provides instructions on how to send Firehose data to Splunk [here in the Terraform documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_firehose_delivery_stream#splunk-destination).

Splunk provides instructions on how to prepare the Splunk platform for Firehose data [here in the Splunk documentation](https://docs.splunk.com/Documentation/AddOns/latest/Firehose/ConfigureFirehose).

### Kinesis

Logs are written to AWS Kinesis streams.

- Plugin name: `kinesis`
- Flag namespace: [kinesis](../02-Deploying/03-Configuration.md#kinesis)

With the Kinesis plugin, osquery result and/or status logs are written to
[Amazon Kinesis Data Streams](https://aws.amazon.com/kinesis/data-streams).

Note that Kinesis logging has limits [discussed in the
documentation](https://docs.aws.amazon.com/kinesis/latest/dev/limits.html).
When Fleet encounters logs that are too big for Kinesis, notifications will be
output in the Fleet logs and those logs _will not_ be sent to Kinesis.

### Lambda

Logs are written to AWS Lambda functions.

- Plugin name: `lambda`
- Flag namespace: [lambda](../02-Deploying/03-Configuration.md#lambda)

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

Logs are written to Google Cloud PubSub topics.

- Plugin name: `pubsub`
- Flag namespace: [pubsub](../02-Deploying/03-Configuration.md#pubsub)

With the PubSub plugin, osquery result and/or status logs are written to [PubSub](https://cloud.google.com/pubsub/) topics.

Note that messages over 10MB will be dropped, with a notification sent to the fleet logs, as these can never be processed by PubSub.

### Stdout

Logs are written to stdout.

- Plugin name: `stdout`
- Flag namespace: [stdout](../02-Deploying/03-Configuration.md#stdout)

With the stdout plugin, osquery result and/or status logs are written to stdout
on the Fleet server. This is typically used for debugging or with a log
forwarding setup that will capture and forward stdout logs into a logging
pipeline. 

Note that if multiple load-balanced Fleet servers are used, the logs
will be load-balanced across those servers (not duplicated).

### Filesystem

Logs are written to the local Fleet server filesystem.

The default log destination.

- Plugin name: `filesystem`
- Flag namespace: [filesystem](../02-Deploying/03-Configuration.md#filesystem)

With the filesystem plugin, osquery result and/or status logs are written to the local filesystem on the Fleet server. This is typically used with a log forwarding agent on the Fleet server that will push the logs into a logging pipeline. 

Note that if multiple load-balanced Fleet servers are used, the logs will be load-balanced across those servers (not duplicated).

### Sending logs outside of Fleet

Osquery agents are typically configured to send logs to the Fleet server (`--logger_plugin=tls`). This is not a requirement, and any other logger plugin can be used even when osquery clients are connecting to the Fleet server to retrieve configuration or run live queries. 

See the [osquery logging documentation](https://osquery.readthedocs.io/en/stable/deployment/logging/) for more about configuring logging on the agent.

If `--logger_plugin=tls` is used with osquery clients, the following configuration can be applied on the Fleet server for handling the incoming logs.
