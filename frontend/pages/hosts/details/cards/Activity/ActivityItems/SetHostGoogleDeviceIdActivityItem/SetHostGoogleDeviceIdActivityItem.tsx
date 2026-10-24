import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const SetHostGoogleDeviceIdActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      set the Google device ID for this host to{" "}
      <b>{activity.details?.google_device_id}</b>.
    </ActivityItem>
  );
};

export default SetHostGoogleDeviceIdActivityItem;
