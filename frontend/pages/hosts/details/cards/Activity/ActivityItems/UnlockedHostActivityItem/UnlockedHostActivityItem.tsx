import React from "react";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";

const baseClass = "unlocked-host-activity-item";

const UnlockedHostActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> unlocked this host.
    </HostActivityItem>
  );
};

export default UnlockedHostActivityItem;
