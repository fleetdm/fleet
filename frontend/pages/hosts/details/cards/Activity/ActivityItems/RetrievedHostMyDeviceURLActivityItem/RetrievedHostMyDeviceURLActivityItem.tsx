import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "retrieved-host-my-device-url-activity-item";

const RetrievedHostMyDeviceURLActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name} </b>
      retrieved the My device URL for this host.
    </ActivityItem>
  );
};

export default RetrievedHostMyDeviceURLActivityItem;
