import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "resent-configuration-profile-activity-item";

const ResentConfigurationProfileActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> resent{" "}
      <b>{activity.details?.profile_name}</b> to this host.
    </ActivityItem>
  );
};

export default ResentConfigurationProfileActivityItem;
