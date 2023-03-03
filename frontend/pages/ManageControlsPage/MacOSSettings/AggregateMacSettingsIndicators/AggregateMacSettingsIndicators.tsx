import { IAggregateMacSettingsStatus } from "interfaces/mdm";
import MacSettingsIndicator from "pages/hosts/details/MacSettingsIndicator";
import React from "react";
import { useQuery } from "react-query";
import paths from "router/paths";
import mdmAPI from "services/entities/mdm";
import { buildQueryStringFromParams } from "utilities/url";

const baseClass = "aggregate-mac-settings-indicators";

interface AggregateMacSettingsIndicatorsProps {
  teamId: number;
}

const AggregateMacSettingsIndicators = ({
  teamId,
}: AggregateMacSettingsIndicatorsProps) => {
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

  const {
    data: aggregateProfileStatusesResponse,
  } = useQuery<IAggregateMacSettingsStatus>(
    ["aggregateProfileStatuses", teamId],
    () => mdmAPI.getAggregateProfileStatuses(teamId)
  );

  const DISPLAY_ORDER = ["latest", "pending", "failing"] as const;
  const orderedResponseKVArr: [
    keyof IAggregateMacSettingsStatus,
    number
  ][] = aggregateProfileStatusesResponse
    ? DISPLAY_ORDER.map((key) => {
        return [key, aggregateProfileStatusesResponse[key]];
      })
    : [];

  const indicators = orderedResponseKVArr.map(([status, count]) => {
    const { text, iconName, tooltipText } = AGGREGATE_STATUS_DISPLAY_OPTIONS[
      status
    ];

    return (
      <div className="aggregate-mac-settings-indicator">
        {/* NOTE - below will be renamed as a general component and moved into the components dir by Gabe */}
        <MacSettingsIndicator
          indicatorText={text}
          iconName={iconName}
          tooltip={{ tooltipText, position: "top" }}
        />
        <a
          href={`${paths.MANAGE_HOSTS}?${buildQueryStringFromParams({
            team_id: teamId,
            macos_settings: status,
          })}`}
        >
          {count} hosts
        </a>
      </div>
    );
  });

  return <div className={baseClass}>{indicators}</div>;
};

export default AggregateMacSettingsIndicators;
