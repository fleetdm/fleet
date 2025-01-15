import React from "react";

import ActivityItem from "components/ActivityItem";
import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "unlocked-host-activity-item";

const UnlockedHostActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  let desc = "unlocked this host.";
  if (activity.details?.host_platform === "darwin") {
    desc = "viewed the six-digit unlock PIN for this host.";
  }
  return (
    <ActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name} </b> {desc}
    </ActivityItem>
  );
};

export default UnlockedHostActivityItem;
