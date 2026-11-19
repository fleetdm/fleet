import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const SetHostGoogleDeviceIdActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const googleEmail = activity.details?.google_email;
  return (
    <ActivityItem activity={activity} hideCancel hideShowDetails>
      <b>{activity.fleet_initiated ? "Fleet" : activity.actor_full_name} </b>
      marked this host as company-owned in Google
      {googleEmail ? (
        <>
          {" "}
          when <b>{googleEmail}</b> signed in.
        </>
      ) : (
        "."
      )}
    </ActivityItem>
  );
};

export default SetHostGoogleDeviceIdActivityItem;
