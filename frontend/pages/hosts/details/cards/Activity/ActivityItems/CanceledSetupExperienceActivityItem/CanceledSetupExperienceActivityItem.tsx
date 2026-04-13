import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-setup-experience-activity-item";

const CanceledSetupExperienceActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <>
        <b>{activity.actor_full_name ?? "Fleet"}</b> canceled setup experience
        on this host because <b>{activity.details.software_title}</b> failed to
        install.
      </>
    </ActivityItem>
  );
};

export default CanceledSetupExperienceActivityItem;
