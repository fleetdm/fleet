import React from "react";

import paths from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { HOSTS_QUERY_PARAMS } from "services/entities/hosts";
import { ProfileStatusSummaryResponse } from "services/entities/mdm";

import Card from "components/Card";
import Spinner from "components/Spinner";
import StatusIndicatorWithIcon, {
  IndicatorStatus,
} from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";
import DataError from "components/DataError";

import AGGREGATE_STATUS_DISPLAY_OPTIONS from "./ProfileStatusAggregateOptions";

const baseClass = "profile-status-aggregate";

interface IProfileStatusCountProps {
  statusIcon: IndicatorStatus;
  title: string;
  hostCount: number;
  tooltipText: JSX.Element;
}

const ProfileStatusCount = ({
  statusIcon,
  title,
  hostCount,
  tooltipText,
}: IProfileStatusCountProps) => {
  return (
    <div className={`${baseClass}__profile-status-count`}>
      <StatusIndicatorWithIcon
        status={statusIcon}
        value={title}
        tooltip={{ tooltipText, position: "top" }}
        layout="vertical"
        valueClassName={`${baseClass}__status-indicator-value`}
      />
      <div className={`${baseClass}__host-count`}>{hostCount} hosts</div>
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

    const hostsByStatusParams = {
      team_id: teamId,
      [HOSTS_QUERY_PARAMS.OS_SETTINGS]: value,
    };

    const path = getPathWithQueryParams(
      paths.MANAGE_HOSTS,
      hostsByStatusParams
    );

    return (
      <Card className={baseClass} borderRadiusSize="large" path={path}>
        <ProfileStatusCount
          key={value}
          statusIcon={iconName}
          title={text}
          hostCount={count}
          tooltipText={tooltipText}
        />
      </Card>
    );
  });

  return <div className={baseClass}>{indicators}</div>;
};

export default ProfileStatusAggregate;
