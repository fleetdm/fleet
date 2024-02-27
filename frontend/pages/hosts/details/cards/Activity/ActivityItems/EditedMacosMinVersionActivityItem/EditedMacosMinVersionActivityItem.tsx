import React from "react";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "edit-macos-min-version-activity-item";

/**
 *
 */
const EditedMacosMinVersionActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  // TODO: This activity also addresses removing the minimum version and deadline
  // from the host, so we need to handle that as well.

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
      <b>{activity.actor_full_name}</b> {statusText} <b>OS updates</b> (minimum
      version: {activity.details?.minimum_version} / deadline:{" "}
      {activity.details?.deadline}) on this host.
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default EditedMacosMinVersionActivityItem;
