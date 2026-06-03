import React from "react";

import ActivityItem from "components/ActivityItem";
import CustomLink from "components/CustomLink";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "failed-wipe-activity-item";

const FailedWipeActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      Wipe failed on this host.
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

export default FailedWipeActivityItem;
