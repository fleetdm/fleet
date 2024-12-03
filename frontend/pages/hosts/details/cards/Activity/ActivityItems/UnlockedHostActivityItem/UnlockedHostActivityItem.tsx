import React from "react";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";

const baseClass = "unlocked-host-activity-item";

const UnlockedHostActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  let desc = "unlocked this host.";
  if (activity.details?.host_platform === "darwin") {
    desc = "viewed the six-digit unlock PIN for this host.";
  }
  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name} </b> {desc}
    </HostActivityItem>
  );
};

export default UnlockedHostActivityItem;
