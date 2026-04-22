import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "disabled-host-orbit-debug-logging-activity-item";

const DisabledHostOrbitDebugLoggingActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> disabled orbit debug logging on this
      host.
    </ActivityItem>
  );
};

export default DisabledHostOrbitDebugLoggingActivityItem;
