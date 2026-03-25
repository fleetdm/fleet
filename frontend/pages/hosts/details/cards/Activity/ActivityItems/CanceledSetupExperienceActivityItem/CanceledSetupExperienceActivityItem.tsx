import React from "react";

import ActivityItem from "components/ActivityItem";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

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
        <b>{activity.actor_full_name}</b> canceled setup experience on this host
        because{" "}
        <b>
          {getDisplayedSoftwareName(
            activity.details.software_title,
            activity.details.software_display_name
          )}
        </b>{" "}
        failed to install.
      </>
    </ActivityItem>
  );
};

export default CanceledSetupExperienceActivityItem;
