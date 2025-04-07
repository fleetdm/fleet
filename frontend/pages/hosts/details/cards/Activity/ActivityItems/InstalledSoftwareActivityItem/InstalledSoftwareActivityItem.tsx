import React from "react";

import { getInstallStatusPredicate } from "interfaces/software";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "installed-software-activity-item";

const InstalledSoftwareActivityItem = ({
  tab,
  activity,
  onShowDetails,
  onCancel,
  hideCancel,
  isSoloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const { actor_full_name: actorName, details } = activity;
  const { self_service, software_title: title } = details;
  const status =
    details.status === "failed" ? "failed_uninstall" : details.status;

  const actorDisplayName = self_service ? (
    <span>End user</span>
  ) : (
    <b>{actorName ?? "Fleet"}</b>
  );

  let installedSoftwarePrefix = getInstallStatusPredicate(status);
  if (tab !== "past" && activity.fleet_initiated) {
    installedSoftwarePrefix =
      status === "pending_uninstall" ? "will uninstall" : "will install";
  }

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      isSoloActivity={isSoloActivity}
    >
      <>{actorDisplayName}</> {installedSoftwarePrefix} <b>{title}</b> on this
      host{self_service && " (self-service)"}.{" "}
    </ActivityItem>
  );
};

export default InstalledSoftwareActivityItem;
