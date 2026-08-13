import React from "react";

import ActivityItem from "components/ActivityItem";
import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const UnlockedUserAccountActivityItem = ({
  activity,
  onShowDetails,
  isSoloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => (
  <ActivityItem
    activity={activity}
    onShowDetails={onShowDetails}
    isSoloActivity={isSoloActivity}
    hideCancel
  >
    <b>{activity.actor_full_name ?? "Fleet"}</b>
    {" unlocked the "}
    <b>{activity.details?.username}</b>
    {" user account on this host."}
  </ActivityItem>
);

export default UnlockedUserAccountActivityItem;
