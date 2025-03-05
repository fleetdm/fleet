import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "locked-host-activity-item";

const LockedHostActivityItem = ({
  activity,
  hideCancel,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
    >
      <b>{activity.actor_full_name}</b> locked this host.
    </ActivityItem>
  );
};

export default LockedHostActivityItem;
