import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const CreatedManagedLocalAccountActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>Fleet </b>
      created a managed local account for this host.
    </ActivityItem>
  );
};

export default CreatedManagedLocalAccountActivityItem;
