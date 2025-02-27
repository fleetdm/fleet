import React from "react";

import paths from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { MdmProfileStatus } from "interfaces/mdm";
import { HOSTS_QUERY_PARAMS } from "services/entities/hosts";
import { ProfileStatusSummaryResponse } from "services/entities/mdm";

import Spinner from "components/Spinner";
import StatusIndicatorWithIcon, {
  IndicatorStatus,
} from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";
import DataError from "components/DataError";

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
  const hostsByStatusParams = {
    team_id: teamId,
    [HOSTS_QUERY_PARAMS.OS_SETTINGS]: statusValue,
  };

  const path = getPathWithQueryParams(paths.MANAGE_HOSTS, hostsByStatusParams);

  return (
    <div className={`${baseClass}__profile-status-count`}>
      <StatusIndicatorWithIcon
        status={statusIcon}
        value={title}
        tooltip={{ tooltipText, position: "top" }}
        layout="vertical"
        valueClassName={`${baseClass}__status-indicator-value`}
      />
      <a href={path}>{hostCount} hosts</a>
    </div>
  );
};

interface ProfileStatusAggregateProps {
  isLoading: boolean;
  isError: boolean;
  teamId: number;
  aggregateProfileStatusData?: ProfileStatusSummaryResponse;
}

const ProfileStatusAggregate = ({
  isLoading,
  isError,
  teamId,
  aggregateProfileStatusData,
}: ProfileStatusAggregateProps) => {
  if (isLoading) {
    return (
      <div className={baseClass}>
        <Spinner className={`${baseClass}__loading-spinner`} centered={false} />
      </div>
    );
  }

  if (isError) {
    return <DataError />;
  }

  if (!aggregateProfileStatusData) return null;

  const indicators = AGGREGATE_STATUS_DISPLAY_OPTIONS.map((status) => {
    const { value, text, iconName, tooltipText } = status;
    const count = aggregateProfileStatusData[value];

    return (
      <ProfileStatusCount
        key={value}
        statusIcon={iconName}
        statusValue={value}
        teamId={teamId}
        title={text}
        hostCount={count}
        tooltipText={tooltipText}
      />
    );
  });

  return <div className={baseClass}>{indicators}</div>;
};

export default ProfileStatusAggregate;
