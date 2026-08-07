import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "installed-all-self-service-software-activity-item";

const InstalledAllSelfServiceSoftwareActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const categoryName = activity.details.self_service_category_name;

  // Self-service install-all can be triggered by anyone who opens the host's My
  // device page, so the actor is dropped in favor of "End user".
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      {categoryName ? (
        <>
          <b>End user</b> selected the <b>Install all</b> option in the
          self-service <b>{categoryName}</b> category.
        </>
      ) : (
        <>
          <b>End user</b> installed all the software in self-service.
        </>
      )}
    </ActivityItem>
  );
};

export default InstalledAllSelfServiceSoftwareActivityItem;
