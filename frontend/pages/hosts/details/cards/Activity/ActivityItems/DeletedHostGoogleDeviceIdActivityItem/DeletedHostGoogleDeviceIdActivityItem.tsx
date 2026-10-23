import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const DeletedHostGoogleDeviceIdActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      deleted the Google device ID <b>
        {activity.details?.google_device_id}
      </b>{" "}
      from this host.
    </ActivityItem>
  );
};

export default DeletedHostGoogleDeviceIdActivityItem;
