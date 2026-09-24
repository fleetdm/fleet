import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "edited-custom-host-vital-value-activity-item";

const EditedCustomHostVitalValueActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> edited the value for custom host vital{" "}
      <b>{activity.details?.custom_host_vital_name}</b>.
    </ActivityItem>
  );
};

export default EditedCustomHostVitalValueActivityItem;
