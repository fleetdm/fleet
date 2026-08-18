import React from "react";

import ActivityItem from "components/ActivityItem";
import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "released-from-ab-activity-item";

const ReleasedFromABActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <>
        <b>{activity.actor_full_name}</b> released this host from Apple
        Business.
      </>
    </ActivityItem>
  );
};

export default ReleasedFromABActivityItem;
