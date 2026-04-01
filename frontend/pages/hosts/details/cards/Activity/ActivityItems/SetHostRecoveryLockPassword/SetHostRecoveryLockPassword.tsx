import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const SetHostRecoveryLockPasswordActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      set a Recovery Lock password for this host.
    </ActivityItem>
  );
};

export default SetHostRecoveryLockPasswordActivityItem;
