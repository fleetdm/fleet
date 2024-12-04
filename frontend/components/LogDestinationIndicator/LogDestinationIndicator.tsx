import React from "react";
import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { read } from "fs";

interface ILogDestinationIndicatorProps {
  logDestination: string;
  excludeTooltip?: boolean;
}

const generateClassTag = (rawValue: string): string => {
  if (rawValue === DEFAULT_EMPTY_CELL_VALUE) {
    return "indeterminate";
  }
  return rawValue.replace(" ", "-").toLowerCase();
};

const LogDestinationIndicator = ({
  logDestination,
  excludeTooltip = false,
}: ILogDestinationIndicatorProps) => {
  const classTag = generateClassTag(logDestination);
  const statusClassName = classnames(
    "log-destination-indicator",
    `log-destination-indicator--${classTag}`,
    `log-destination--${classTag}`
  );
  const readableLogDestination = () => {
    switch (logDestination) {
      case "filesystem":
        return "Filesystem";
      case "firehose":
        return "Amazon Kinesis Data Firehose";
      case "kinesis":
        return "Amazon Kinesis Data Streams";
      case "lambda":
        return "AWS Lambda";
      case "pubsub":
        return "Google Cloud Pub/Sub";
      case "kafta":
        return "Apache Kafka";
      case "stdout":
        return "Standard output (stdout)";
      case "":
        return "Not configured";
      default:
        return logDestination;
    }
  };

  const tooltipText = () => {
    switch (logDestination) {
      case "filesystem":
        return (
          <>
            Each time a query runs, the data is sent to <br />
            /var/log/osquery/osqueryd.snapshots.log <br />
            in each host&apos;s filesystem.
          </>
        );
      case "firehose":
        return (
          <>
            Each time a query runs, the data is sent to <br />
            Amazon Kinesis Data Firehose.
          </>
        );
      case "kinesis":
        return (
          <>
            Each time a query runs, the data is sent to <br />
            Amazon Kinesis Data Streams.
          </>
        );
      case "lambda":
        return (
          <>
            Each time a query runs, the data <br />
            is sent to AWS Lambda.
          </>
        );
      case "pubsub":
        return (
          <>
            Each time a query runs, the data is <br /> sent to Google Cloud Pub
            / Sub.
          </>
        );
      case "kafta":
        return (
          <>
            Each time a query runs, the data <br /> is sent to Apache Kafka.
          </>
        );
      case "stdout":
        return (
          <>
            Each time a query runs, the data is sent to <br />
            standard output(stdout) on the Fleet server.
          </>
        );
      case "":
        return <>Please configure a log destination.</>;
      default:
        return (
          <>
            No additional information is available about this log destination.
          </>
        );
    }
  };

  return excludeTooltip ? (
    readableLogDestination()
  ) : (
    <TooltipWrapper tipContent={tooltipText()} className={statusClassName}>
      {readableLogDestination()}
    </TooltipWrapper>
  );
};

export default LogDestinationIndicator;
