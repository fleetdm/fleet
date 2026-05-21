import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "resent-certificate-activity-item";

const ResentCertificateActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <b>{activity.actor_full_name}</b> resent{" "}
      <b>{activity.details?.certificate_name}</b> certificate on this host.
    </ActivityItem>
  );
};

export default ResentCertificateActivityItem;
