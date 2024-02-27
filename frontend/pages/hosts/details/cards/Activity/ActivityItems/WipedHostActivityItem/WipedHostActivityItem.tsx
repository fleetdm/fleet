import React from "react";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "wiped-host-activity-item";

const WipedHostActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  let statusText;

  switch (activity.details?.status) {
    case "Acknowledged":
      statusText = "wiped";
      break;
    case "Pending":
      statusText = "told Fleet to wipe";
      break;
    case "Failed":
      statusText = "failed to wipe";
      break;
    default:
  }

  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> {statusText} this host.{" "}
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default WipedHostActivityItem;
