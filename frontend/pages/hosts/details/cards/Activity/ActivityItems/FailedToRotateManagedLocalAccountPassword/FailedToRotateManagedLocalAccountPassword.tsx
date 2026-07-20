import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const FailedToRotateManagedLocalAccountPasswordActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>Fleet </b>
      failed to rotate the managed local account password for this host.
    </ActivityItem>
  );
};

export default FailedToRotateManagedLocalAccountPasswordActivityItem;
