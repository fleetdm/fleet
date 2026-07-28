import React from "react";

import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "mdm-enrolled-activity-item";

const MdmEnrolledActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const { actor_full_name } = activity;
  const platform = activity.details?.platform ?? "";

  let content: React.ReactNode;
  if (isAndroid(platform) || isIPadOrIPhone(platform)) {
    content = actor_full_name ? (
      <>
        <b>{actor_full_name}</b> told Fleet to enroll this host.
      </>
    ) : (
      <>This host enrolled to Fleet.</>
    );
  } else {
    content = actor_full_name ? (
      <>
        <b>{actor_full_name}</b> told Fleet to turn on mobile device management
        (MDM) for this host.
      </>
    ) : (
      <>Mobile device management (MDM) was turned on for this host.</>
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

export default MdmEnrolledActivityItem;
