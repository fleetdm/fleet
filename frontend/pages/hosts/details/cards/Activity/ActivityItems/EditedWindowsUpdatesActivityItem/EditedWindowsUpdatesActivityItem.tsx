import React from "react";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "edited-windows-updates-activity-item";

const EditedWindowsUpdatesActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  let statusText;

  switch (activity.details?.status) {
    case "Acknowledged":
      statusText = "enforced";
      break;
    case "Pending":
      statusText = "told Fleet to enforce";
      break;
    case "Failed":
      statusText = "failed to enforce";
      break;
    default:
  }

  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> {statusText} <b>OS updates</b>{" "}
      (deadline: {activity.details?.deadline_days} / grace period:{" "}
      {activity.details?.grace_period_days}) on this host.
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default EditedWindowsUpdatesActivityItem;
