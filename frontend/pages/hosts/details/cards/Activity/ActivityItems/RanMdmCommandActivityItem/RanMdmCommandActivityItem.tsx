import React from "react";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import HostActivityItem from "../../HostActivityItem";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "ran-mdm-command-activity-item";

const RanMdmCommandActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> told Fleet to run an MDM command on this
      host.{" "}
      <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
    </HostActivityItem>
  );
};

export default RanMdmCommandActivityItem;
