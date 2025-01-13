import React from "react";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";
import HostActivityItem from "../../../../../../../components/ActivityItem";

const baseClass = "locked-host-activity-item";

const LockedHostActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> locked this host.
    </HostActivityItem>
  );
};

export default LockedHostActivityItem;
