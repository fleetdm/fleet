import StatusIndicator from "components/StatusIndicator";
import React from "react";

interface IQueryAutomationsStatusIndicator {
  automationsEnabled: boolean;
  interval: number;
}

enum QueryAutomationsStatus {
  ON = "On",
  OFF = "Off",
  PAUSED = "Paused",
}

const QueryAutomationsStatusIndicator = ({
  automationsEnabled,
  interval,
}: IQueryAutomationsStatusIndicator) => {
  let status;
  if (automationsEnabled) {
    if (interval === 0) {
      status = QueryAutomationsStatus.PAUSED;
    } else {
      status = QueryAutomationsStatus.ON;
    }
  } else {
    status = QueryAutomationsStatus.OFF;
  }

  const tooltip =
    status === QueryAutomationsStatus.PAUSED
      ? {
          tooltipText: (
            <>
              <strong>Automations</strong> will resume for this query when a
              frequency is set.
            </>
          ),
        }
      : undefined;
  return (
    <StatusIndicator
      value={status}
      tooltip={tooltip}
      customIndicatorType={"query-automations"}
    />
  );
};

export default QueryAutomationsStatusIndicator;
