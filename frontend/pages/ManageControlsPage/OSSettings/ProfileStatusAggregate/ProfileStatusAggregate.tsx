import React from "react";

import paths from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { MdmProfileStatus } from "interfaces/mdm";
import { ProfileStatusSummaryResponse } from "services/entities/mdm";

import Spinner from "components/Spinner";
import StatusIndicatorWithIcon, {
  IndicatorStatus,
} from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";

import AGGREGATE_STATUS_DISPLAY_OPTIONS from "./ProfileStatusAggregateOptions";

const baseClass = "profile-status-aggregate";

interface IProfileStatusCountProps {
  statusIcon: IndicatorStatus;
  statusValue: MdmProfileStatus;
  title: string;
  teamId: number;
  hostCount: number;
  tooltipText: string;
}

const ProfileStatusCount = ({
  statusIcon,
  statusValue,
  teamId,
  title,
  hostCount,
  tooltipText,
}: IProfileStatusCountProps) => {
  const generateFilterHostsByStatusLink = () => {
    return `${paths.MANAGE_HOSTS}?${buildQueryStringFromParams({
      team_id: teamId,
      macos_settings: statusValue,
    })}`;
  };

  return (
    <div className={`${baseClass}__profile-status-count`}>
      <StatusIndicatorWithIcon
        status={statusIcon}
        value={title}
        tooltip={{ tooltipText, position: "top" }}
        layout="vertical"
        valueClassName={`${baseClass}__status-indicator-value`}
      />
      <a href={generateFilterHostsByStatusLink()}>{hostCount} hosts</a>
    </div>
  );
};

interface ProfileStatusAggregateProps {
  isLoading: boolean;
  teamId: number;
  aggregateProfileStatusData?: ProfileStatusSummaryResponse;
}

const ProfileStatusAggregate = ({
  isLoading,
  teamId,
  aggregateProfileStatusData,
}: ProfileStatusAggregateProps) => {
  const indicators = AGGREGATE_STATUS_DISPLAY_OPTIONS.map((status) => {
    if (!aggregateProfileStatusData) return null;

    const { value, text, iconName, tooltipText } = status;
    const count = aggregateProfileStatusData[value];

    return (
      <ProfileStatusCount
        statusIcon={iconName}
        statusValue={value}
        teamId={teamId}
        title={text}
        hostCount={count}
        tooltipText={tooltipText}
      />
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

export default ProfileStatusAggregate;
