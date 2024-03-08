import StatusIndicator from "components/StatusIndicator";
import React from "react";

interface IQueryAutomationsStatusIndicator {
  automationsEnabled: boolean;
  interval: number;
}

const QueryAutomationsStatusIndicator = ({
  automationsEnabled,
  interval,
}: IQueryAutomationsStatusIndicator) => {
  let status;
  if (automationsEnabled) {
    if (interval === 0) {
      status = "paused";
    } else {
      status = "on";
    }
  } else {
    status = "off";
  }

  const tooltip =
    status === "paused"
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
      customIndicatorType="query-automations"
    />
  );
};

export default QueryAutomationsStatusIndicator;
