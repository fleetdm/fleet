import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "reset-policy-activity-item";

const ResetPolicyActivityItem = ({
  activity,
  isSoloActivity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      isSoloActivity={isSoloActivity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> reset the policy{" "}
      <b>{activity.details?.policy_name}</b> for this host.
    </ActivityItem>
  );
};

export default ResetPolicyActivityItem;
