import { IconNames } from "components/icons";
import { MdmProfileStatus } from "interfaces/mdm";
import MacSettingsIndicator from "pages/hosts/details/MacSettingsIndicator";
import React from "react";
import { useQuery } from "react-query";
import paths from "router/paths";
import mdmAPI from "services/entities/mdm";
import { buildQueryStringFromParams } from "utilities/url";

const baseClass = "aggregate-mac-settings-indicators";

interface IAggregateDisplayOption {
  value: MdmProfileStatus;
  text: string;
  iconName: IconNames;
  tooltipText: string;
}

const AGGREGATE_STATUS_DISPLAY_OPTIONS: IAggregateDisplayOption[] = [
  {
    value: MdmProfileStatus.VERIFYING,
    text: "Verifying",
    iconName: "success-partial",
    tooltipText:
      "Hosts that told Fleet all settings are enforced. Fleet is verifying.",
  },
  {
    value: MdmProfileStatus.PENDING,
    text: "Pending",
    iconName: "pending",
    tooltipText:
      "Hosts that haven’t applied the latest settings because they are asleep, disconnected from the internet, or require action.",
  },
  {
    value: MdmProfileStatus.FAILED,
    text: "Failed",
    iconName: "error",
    tooltipText:
      "Hosts that failed to apply the latest settings. View hosts to see errors.",
  },
];

type ProfileSummaryResponse = Record<MdmProfileStatus, number>;

interface AggregateMacSettingsIndicatorsProps {
  teamId: number;
}

const AggregateMacSettingsIndicators = ({
  teamId,
}: AggregateMacSettingsIndicatorsProps) => {
  const {
    data: aggregateProfileStatusesResponse,
  } = useQuery<ProfileSummaryResponse>(
    ["aggregateProfileStatuses", teamId],
    () => mdmAPI.getAggregateProfileStatuses(teamId),
    { refetchOnWindowFocus: false }
  );

  if (!aggregateProfileStatusesResponse) return null;

  const indicators = AGGREGATE_STATUS_DISPLAY_OPTIONS.map((status) => {
    const { value, text, iconName, tooltipText } = status;
    const count = aggregateProfileStatusesResponse[value];

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

  return <div className={baseClass}>{indicators}</div>;
};

export default AggregateMacSettingsIndicators;
