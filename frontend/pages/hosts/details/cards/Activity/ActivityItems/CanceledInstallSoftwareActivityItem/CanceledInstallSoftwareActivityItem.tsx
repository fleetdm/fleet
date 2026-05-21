import React from "react";

import ActivityItem from "components/ActivityItem";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-install-software-activity-item";

const CanceledInstallSoftwareActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const fromSetupExperience = activity.details?.from_setup_experience;

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <>
        <b>{activity.actor_full_name}</b> canceled{" "}
        <b>
          {getDisplayedSoftwareName(
            activity.details.software_title,
            activity.details.software_display_name
          )}
        </b>{" "}
        install on this host
        {fromSetupExperience ? " during setup experience" : ""}.
      </>
    </ActivityItem>
  );
};

export default CanceledInstallSoftwareActivityItem;
