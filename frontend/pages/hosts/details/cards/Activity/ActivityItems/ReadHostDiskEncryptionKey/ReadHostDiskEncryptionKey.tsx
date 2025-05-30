import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "locked-host-activity-item";

const ReadHostDiskEncryptionKeyActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name} </b>
      viewed the disk encryption key for this host.
    </ActivityItem>
  );
};

export default ReadHostDiskEncryptionKeyActivityItem;
