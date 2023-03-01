import { IAggregateMacSettingsStatus } from "interfaces/mdm";
import MacSettingsIndicator from "pages/hosts/details/MacSettingsIndicator";
import React from "react";

const baseClass = "aggregate-mac-settings";

interface IAggregateMacSettingsProps {
  aggregateProfileData: IAggregateMacSettingsStatus;
}

const AggregateMacSettingsIndicators = ({
  aggregateProfileData,
}: IAggregateMacSettingsProps) => {
  const AGGREGATE_STATUS_DISPLAY_OPTIONS = {
    latest: {
      text: "Latest",
      iconName: "success",
      tooltipText: "Hosts that applied the latest settings.",
    },
    pending: {
      text: "Pending",
      iconName: "pending",
      tooltipText:
        "Hosts that haven’t applied the latest settings because they are asleep, disconnected from the internet, or require action.",
    },
    failing: {
      text: "Failing",
      iconName: "error",
      tooltipText:
        "Hosts that failed to apply the latest settings. View hosts to see errors.",
    },
  } as const;

  // TODO - typescript doesn't like this map, so spread it out
  const indicators = Object.entries(aggregateProfileData).map(
    (indicatorType, count) => {
      const { text, iconName, tooltipText } = AGGREGATE_STATUS_DISPLAY_OPTIONS[
        // TODO: clean up this typing
        (indicatorType as unknown) as keyof IAggregateMacSettingsStatus
      ];
      return (
        <div>
          <MacSettingsIndicator
            indicatorText={text}
            iconName={iconName}
            tooltip={{ tooltipText, position: "top" }}
          />
          <a href="TODO">{count} hosts</a>
        </div>
      );
    }
  );

  return <div className={baseClass}>{indicators}</div>;
};

export default AggregateMacSettingsIndicators;
