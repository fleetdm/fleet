import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const CreatedManagedLocalAccountActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>Fleet </b>
      created the managed account for this host.
    </ActivityItem>
  );
};

export default CreatedManagedLocalAccountActivityItem;
