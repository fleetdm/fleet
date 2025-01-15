import React from "react";

import { getInstallStatusPredicate } from "interfaces/software";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "installed-software-activity-item";

const InstalledSoftwareActivityItem = ({
  activity,
  onShowDetails,
  hideCancel,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const { actor_full_name: actorName, details } = activity;
  const { self_service, software_title: title } = details;
  const status =
    details.status === "failed" ? "failed_uninstall" : details.status;

  const actorDisplayName = self_service ? (
    <span>End user</span>
  ) : (
    <b>{actorName}</b>
  );

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
      onShowDetails={onShowDetails}
    >
      <>{actorDisplayName}</> {getInstallStatusPredicate(status)} <b>{title}</b>{" "}
      on this host {self_service && "(self-service)"}.{" "}
    </ActivityItem>
  );
};

export default InstalledSoftwareActivityItem;
