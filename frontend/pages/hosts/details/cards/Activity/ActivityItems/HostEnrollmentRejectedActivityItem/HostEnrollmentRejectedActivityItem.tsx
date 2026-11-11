import React from "react";

import ActivityItem from "components/ActivityItem";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "host-enrollment-rejected-activity-item";

const HostEnrollmentRejectedActivityItem = ({
  activity,
  onShowDetails,
  isSoloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      onShowDetails={onShowDetails}
      isSoloActivity={isSoloActivity}
      hideCancel
    >
      {activity.details?.reason === "end_user_authentication_required" ? (
        <span>
          <b>Fleet</b> rejected an automatic enrollment for this host because
          IdP authentication is required.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/rejected-automatic-enrollment`}
            text="Learn more"
            newTab
          />
        </span>
      ) : (
        <span>
          <b>Fleet</b> rejected an enrollment for this host.
        </span>
      )}
    </ActivityItem>
  );
};

export default HostEnrollmentRejectedActivityItem;
