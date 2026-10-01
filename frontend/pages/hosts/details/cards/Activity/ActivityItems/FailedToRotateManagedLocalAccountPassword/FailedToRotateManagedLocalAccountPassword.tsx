import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const FailedToRotateManagedLocalAccountPasswordActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  // Gated on the reason rather than on the platform: only Windows reports one, and a Windows failure without one is
  // possible, so this shows the icon exactly when there is something behind it.
  const hasDetail = !!activity.details?.detail;

  return (
    <ActivityItem
      activity={activity}
      hideCancel
      hideShowDetails={!hasDetail}
      onShowDetails={onShowDetails}
    >
      <b>Fleet </b>
      failed to rotate the managed local account password for this host.
    </ActivityItem>
  );
};

export default FailedToRotateManagedLocalAccountPasswordActivityItem;
