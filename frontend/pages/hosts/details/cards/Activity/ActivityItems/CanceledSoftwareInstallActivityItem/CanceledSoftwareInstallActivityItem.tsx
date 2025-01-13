import React from "react";

import HostActivityItem from "../../../../../../../components/ActivityItem";
import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-software-install-activity-item";

const CanceledSoftwareInstallActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <>
        <b>{activity.actor_full_name}</b> canceled{" "}
        <b>{activity.details?.software_title}</b> install on this host.
      </>
    </HostActivityItem>
  );
};

export default CanceledSoftwareInstallActivityItem;
