import React from "react";

import ActivityItem from "components/ActivityItem";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-uninstall-software-activty-item";

const CanceledUninstallSoftwareActivtyItem = ({
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
        <b>{activity.actor_full_name}</b> canceled{" "}
        <b>
          {getDisplayedSoftwareName(
            activity.details.software_title,
            activity.details.software_display_name
          )}
        </b>{" "}
        uninstall on this host.
      </>
    </ActivityItem>
  );
};

export default CanceledUninstallSoftwareActivtyItem;
