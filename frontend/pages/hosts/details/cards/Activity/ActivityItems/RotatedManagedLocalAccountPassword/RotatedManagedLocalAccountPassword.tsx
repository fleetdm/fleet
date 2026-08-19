import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const RotatedManagedLocalAccountPasswordActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      triggered rotation of the managed local account password for this host.
    </ActivityItem>
  );
};

export default RotatedManagedLocalAccountPasswordActivityItem;
