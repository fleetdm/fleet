import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const DeletedHostGoogleDeviceIdActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.actor_full_name} </b>
      unmarked this host as company-owned in Google.
    </ActivityItem>
  );
};

export default DeletedHostGoogleDeviceIdActivityItem;
