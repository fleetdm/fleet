import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const FailedToRotateDiskEncryptionKeyActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  // Details are offered even without a recorded reason: the modal also tells the
  // admin how the end user recovers.
  return (
    <ActivityItem activity={activity} hideCancel onShowDetails={onShowDetails}>
      <b>Fleet </b>
      failed to rotate the disk encryption key for this host.
    </ActivityItem>
  );
};

export default FailedToRotateDiskEncryptionKeyActivityItem;
