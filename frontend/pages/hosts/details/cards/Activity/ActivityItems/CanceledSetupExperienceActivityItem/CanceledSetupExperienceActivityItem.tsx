import React from "react";

import ActivityItem from "components/ActivityItem";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-setup-experience-activity-item";

const CanceledSetupExperienceActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const title = getDisplayedSoftwareName(
    activity.details.software_title,
    activity.details.software_display_name
  );
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <>
        <b>{activity.actor_full_name ?? "Fleet"}</b> canceled setup experience
        on this host because <b>{title}</b> failed to install. End user was
        asked to restart.
      </>
    </ActivityItem>
  );
};

export default CanceledSetupExperienceActivityItem;
