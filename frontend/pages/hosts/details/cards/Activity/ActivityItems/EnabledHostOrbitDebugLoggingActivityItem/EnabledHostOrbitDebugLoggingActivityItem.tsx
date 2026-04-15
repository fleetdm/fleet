import React from "react";

import ActivityItem from "components/ActivityItem";
import { internationalTimeFormat } from "utilities/helpers";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "enabled-host-orbit-debug-logging-activity-item";

const EnabledHostOrbitDebugLoggingActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const expiresAt = activity.details?.expires_at;
  const expiresSuffix = expiresAt
    ? ` until ${internationalTimeFormat(new Date(expiresAt))}`
    : "";

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> enabled orbit debug logging on this host
      {expiresSuffix}.
    </ActivityItem>
  );
};

export default EnabledHostOrbitDebugLoggingActivityItem;
