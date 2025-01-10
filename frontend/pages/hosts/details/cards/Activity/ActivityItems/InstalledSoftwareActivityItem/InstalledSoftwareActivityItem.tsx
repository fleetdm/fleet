import React from "react";

import { getInstallStatusPredicate } from "interfaces/software";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";

const baseClass = "installed-software-activity-item";

const InstalledSoftwareActivityItem = ({
  activity,
  onShowDetails,
  hideClose,
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
    <HostActivityItem
      className={baseClass}
      activity={activity}
      hideClose={hideClose}
      onShowDetails={onShowDetails}
    >
      <>{actorDisplayName}</> {getInstallStatusPredicate(status)} <b>{title}</b>{" "}
      on this host {self_service && "(self-service)"}.{" "}
    </HostActivityItem>
  );
};

export default InstalledSoftwareActivityItem;
