import { IAggregateMacSettingsStatus } from "interfaces/mdm";
import MacSettingsIndicator from "pages/hosts/details/MacSettingsIndicator";
import React from "react";

const baseClass = "aggregate-mac-settings-indicators";

const AggregateMacSettingsIndicators = () => {
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

  const aggregateProfileData: IAggregateMacSettingsStatus = {
    latest: 100,
    pending: 100,
    failing: 100,
  };

  const aggregateStatusDataArray = Object.entries(aggregateProfileData) as [
    keyof IAggregateMacSettingsStatus,
    number
  ][];

  const indicators = aggregateStatusDataArray.map(([indicatorType, count]) => {
    const { text, iconName, tooltipText } = AGGREGATE_STATUS_DISPLAY_OPTIONS[
      indicatorType
    ];
    return (
      <div className="aggregate-mac-settings-indicator">
        {/* NOTE - below component will be renamed GenericStatusIndicator and moved into the components dir by Gabe */}
        <MacSettingsIndicator
          indicatorText={text}
          iconName={iconName}
          tooltip={{ tooltipText, position: "top" }}
        />
        <a href="TODO">{count} hosts</a>
      </div>
    );
  });

  return <div className={baseClass}>{indicators}</div>;
};

export default AggregateMacSettingsIndicators;
