import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const CreatedDiskEncryptionPINActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>End user </b>
      created a disk encryption PIN for this host.
    </ActivityItem>
  );
};

export default CreatedDiskEncryptionPINActivityItem;
