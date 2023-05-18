import React from "react";

import paths from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { MdmProfileStatus, ProfileSummaryResponse } from "interfaces/mdm";
import MacSettingsIndicator from "pages/hosts/details/MacSettingsIndicator";

import { IconNames } from "components/icons";
import Spinner from "components/Spinner";

const baseClass = "aggregate-mac-settings-indicators";

interface IAggregateDisplayOption {
  value: MdmProfileStatus;
  text: string;
  iconName: IconNames;
  tooltipText: string;
}

const AGGREGATE_STATUS_DISPLAY_OPTIONS: IAggregateDisplayOption[] = [
  {
    value: "verifying",
    text: "Verifying",
    iconName: "success-partial",
    tooltipText:
      "Hosts that told Fleet all settings are enforced. Fleet is verifying.",
  },
  {
    value: "pending",
    text: "Pending",
    iconName: "pending-partial",
    tooltipText:
      "Hosts that will have settings enforced when the hosts come online.",
  },
  {
    value: "failed",
    text: "Failed",
    iconName: "error",
    tooltipText:
      "Hosts that failed to apply settings. Click on a host to view error(s).",
  },
];

interface AggregateMacSettingsIndicatorsProps {
  isLoading: boolean;
  teamId: number;
  aggregateProfileStatusData?: ProfileSummaryResponse;
}

const AggregateMacSettingsIndicators = ({
  isLoading,
  teamId,
  aggregateProfileStatusData,
}: AggregateMacSettingsIndicatorsProps) => {
  const indicators = AGGREGATE_STATUS_DISPLAY_OPTIONS.map((status) => {
    if (!aggregateProfileStatusData) return null;

    const { value, text, iconName, tooltipText } = status;
    const count = aggregateProfileStatusData[value];

    return (
      <div className="aggregate-mac-settings-indicator">
        <MacSettingsIndicator
          indicatorText={text}
          iconName={iconName}
          tooltip={{ tooltipText, position: "top" }}
        />
        <a
          href={`${paths.MANAGE_HOSTS}?${buildQueryStringFromParams({
            team_id: teamId,
            macos_settings: value,
          })}`}
        >
          {count} hosts
        </a>
      </div>
    );
  });

  if (isLoading) {
    return (
      <div className={baseClass}>
        <Spinner className={`${baseClass}__loading-spinner`} centered={false} />
      </div>
    );
  }

  return <div className={baseClass}>{indicators}</div>;
};

export default AggregateMacSettingsIndicators;
