import React from "react";

import ActivityItem from "components/ActivityItem";
import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-software-install-activity-item";

const CanceledSoftwareInstallActivityItem = ({
  activity,
  hideCancel,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
      hideShowDetails
    >
      <>
        <b>{activity.actor_full_name}</b> canceled{" "}
        <b>{activity.details?.software_title}</b> install on this host.
      </>
    </ActivityItem>
  );
};

export default CanceledSoftwareInstallActivityItem;
