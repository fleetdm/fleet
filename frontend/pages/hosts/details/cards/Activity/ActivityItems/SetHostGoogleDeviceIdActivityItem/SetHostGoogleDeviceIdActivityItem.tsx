import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const SetHostGoogleDeviceIdActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      marked this host as company-owned in Google.
    </ActivityItem>
  );
};

export default SetHostGoogleDeviceIdActivityItem;
