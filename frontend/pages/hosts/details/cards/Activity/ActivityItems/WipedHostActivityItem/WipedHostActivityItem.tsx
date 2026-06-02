import React from "react";

import ActivityItem from "components/ActivityItem";
import CustomLink from "components/CustomLink";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "wiped-host-activity-item";

const WipedHostActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> wiped this host.
      {activity.details?.host_platform === "linux" && (
        <>
          {" "}
          <CustomLink
            url="https://fleetdm.com/guides/lock-wipe-hosts"
            text="Learn more"
            newTab
          />{" "}
          Expecting it to work differently?{" "}
          <CustomLink
            url="https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title="
            text="File a feature request"
            newTab
          />
        </>
      )}
    </ActivityItem>
  );
};

export default WipedHostActivityItem;
