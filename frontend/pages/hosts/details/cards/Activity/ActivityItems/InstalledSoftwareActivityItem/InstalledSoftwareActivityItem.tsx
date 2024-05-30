import React from "react";

import { SoftwareInstallStatus } from "interfaces/software";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "installed-software-activity-item";

const STATUS_PREDICATES: Record<SoftwareInstallStatus, string> = {
  failed: "failed to install",
  installed: "installed",
  pending: "told Fleet to install",
} as const;

export const getSoftwareInstallStatusPredicate = (
  status: string | undefined
) => {
  if (!status) {
    return STATUS_PREDICATES.pending;
  }
  return (
    STATUS_PREDICATES[status as SoftwareInstallStatus] ||
    STATUS_PREDICATES.pending
  );
};

const InstalledSoftwareActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const { actor_full_name: actorName, details } = activity;
  const { self_service, status, software_title: title } = details;

  const actorDisplayName = self_service ? (
    <span>An end user</span>
  ) : (
    <b>{actorName}</b>
  );

  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <>{actorDisplayName}</> {getSoftwareInstallStatusPredicate(status)}{" "}
      <b>{title}</b> software on this host.{" "}
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default InstalledSoftwareActivityItem;
