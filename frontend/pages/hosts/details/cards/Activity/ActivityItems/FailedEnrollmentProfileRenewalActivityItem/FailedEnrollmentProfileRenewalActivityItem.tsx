import React from "react";

import ActivityItem from "components/ActivityItem";
import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "failed-enrollment-profile-renewal-activity-item";

const FailedEnrollmentProfileRenewalActivityItem = ({
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
        <b>Fleet</b> enrollment profile renewal failed for this host.{" "}
      </span>
    </ActivityItem>
  );
};

export default FailedEnrollmentProfileRenewalActivityItem;
