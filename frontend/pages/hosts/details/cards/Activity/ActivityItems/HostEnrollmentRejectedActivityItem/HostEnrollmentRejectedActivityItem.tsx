import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "host-enrollment-rejected-activity-item";

const HostEnrollmentRejectedActivityItem = ({
  activity,
  onShowDetails,
  isSoloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      onShowDetails={onShowDetails}
      isSoloActivity={isSoloActivity}
      hideCancel
    >
      <span>
        <b>Fleet</b> rejected an enrollment for this host.
      </span>
    </ActivityItem>
  );
};

export default HostEnrollmentRejectedActivityItem;
