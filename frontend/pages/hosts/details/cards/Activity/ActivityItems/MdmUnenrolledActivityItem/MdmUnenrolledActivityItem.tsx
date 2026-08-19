import React from "react";

import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "mdm-unenrolled-activity-item";

const MdmUnenrolledActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const { actor_full_name } = activity;
  const platform = activity.details?.platform ?? "";

  let content: React.ReactNode;
  if (isAndroid(platform) || isIPadOrIPhone(platform)) {
    content = actor_full_name ? (
      <>
        <b>{actor_full_name}</b> told Fleet to unenroll this host.
      </>
    ) : (
      <>This host is unenrolled from Fleet.</>
    );
  } else {
    content = actor_full_name ? (
      <>
        <b>{actor_full_name}</b> told Fleet to turn off mobile device management
        (MDM) for this host.
      </>
    ) : (
      <>Mobile device management (MDM) was turned off for this host.</>
    );
  }

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      {content}
    </ActivityItem>
  );
};

export default MdmUnenrolledActivityItem;
