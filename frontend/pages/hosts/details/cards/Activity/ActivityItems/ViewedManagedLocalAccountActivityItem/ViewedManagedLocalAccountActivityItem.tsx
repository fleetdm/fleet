import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const ViewedManagedLocalAccountActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      viewed the managed local account on this host.
    </ActivityItem>
  );
};

export default ViewedManagedLocalAccountActivityItem;
