import React from "react";

import ActivityItem from "components/ActivityItem";
import { ActivityType } from "interfaces/activity";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "opt-in-configuration-profile-activity-item";

const OptInConfigurationProfileActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const verb =
    activity.type === ActivityType.InstalledOptInConfigurationProfile
      ? "installed"
      : "uninstalled";
  const actor = activity.details?.self_service
    ? "End user"
    : activity.actor_full_name || "Fleet";

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{actor}</b> {verb} the opt-in <b>{activity.details?.profile_name}</b>{" "}
      profile on this host.
    </ActivityItem>
  );
};

export default OptInConfigurationProfileActivityItem;
