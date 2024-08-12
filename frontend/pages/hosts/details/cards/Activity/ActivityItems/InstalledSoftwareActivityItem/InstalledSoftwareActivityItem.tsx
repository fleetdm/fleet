import React from "react";

import { getInstallStatusPredicate } from "interfaces/software";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "installed-software-activity-item";

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
      <>{actorDisplayName}</> {getInstallStatusPredicate(status)} <b>{title}</b>{" "}
      on this host.{" "}
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default InstalledSoftwareActivityItem;
