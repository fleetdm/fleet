import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-mdm-command-activity-item";

const CanceledMdmCommandActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <>
        <b>{activity.actor_full_name}</b> canceled the pending{" "}
        <b>{activity.details.command_type}</b> command on this host.
      </>
    </ActivityItem>
  );
};

export default CanceledMdmCommandActivityItem;
