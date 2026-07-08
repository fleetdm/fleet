import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "connected-to-host-activity-item";

const ConnectedToHostActivityItem = ({
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
      connected to this host.
    </ActivityItem>
  );
};

export default ConnectedToHostActivityItem;
