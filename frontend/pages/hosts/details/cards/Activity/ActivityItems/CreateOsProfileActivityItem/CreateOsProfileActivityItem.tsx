import React from "react";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "create-os-profile-activity-item";

const CreateOsProfileActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  let statusText;

  switch (activity.details?.status) {
    case "Acknowledged":
      statusText = "added";
      break;
    case "Pending":
      statusText = "told Fleet to add";
      break;
    case "Failed":
      statusText = "failed to add";
      break;
    default:
  }

  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> {statusText} configuration profile
      Restrictions to this host.
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default CreateOsProfileActivityItem;
