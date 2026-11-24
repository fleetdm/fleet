import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const ViewedHostUnlockPinActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const hostName = activity.details?.host_display_name;
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      viewed the unlock PIN for {hostName ? <b>{hostName}</b> : "this host"}.
    </ActivityItem>
  );
};

export default ViewedHostUnlockPinActivityItem;
