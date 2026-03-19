import React from "react";

import { ActivityType } from "interfaces/activity";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const RotatedHostRecoveryLockPasswordActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const isAutoRotation =
    activity.type === ActivityType.AutoRotatedHostRecoveryLockPassword;
  const actionText = isAutoRotation
    ? "auto-rotated the Recovery Lock password for this host."
    : "triggered a Recovery Lock password rotation for this host.";

  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      {actionText}
    </ActivityItem>
  );
};

export default RotatedHostRecoveryLockPasswordActivityItem;
