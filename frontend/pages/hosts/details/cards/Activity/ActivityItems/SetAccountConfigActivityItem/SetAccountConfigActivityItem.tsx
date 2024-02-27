import React from "react";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "set-account-config-activity-item";

const SetAccountConfigActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  let statusText;

  switch (activity.details?.status) {
    case "Acknowledged":
      statusText = "ran";
      break;
    case "Pending":
      statusText = "will run";
      break;
    case "Failed":
      statusText = "failed to run";
      break;
    default:
  }

  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> {statusText} the MDM command to set the
      <b>full name</b> and <b>account name</b> for the end user&apos;s local
      account on this host
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default SetAccountConfigActivityItem;
