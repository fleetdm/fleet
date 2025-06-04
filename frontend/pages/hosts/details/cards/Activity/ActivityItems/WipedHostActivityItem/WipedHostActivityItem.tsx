import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "wiped-host-activity-item";

const WipedHostActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> wiped this host.
    </ActivityItem>
  );
};

export default WipedHostActivityItem;
