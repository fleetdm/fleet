import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const FailedToRotateDiskEncryptionKeyActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const hasDetail = !!activity.details?.detail;

  return (
    <ActivityItem
      activity={activity}
      hideCancel
      hideShowDetails={!hasDetail}
      onShowDetails={onShowDetails}
    >
      <b>Fleet </b>
      failed to rotate the disk encryption key for this host.
    </ActivityItem>
  );
};

export default FailedToRotateDiskEncryptionKeyActivityItem;
