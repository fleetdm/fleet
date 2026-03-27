import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "installed-certificate-activity-item";

const InstalledCertificateActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  const isFailed = activity.details?.status === "failed_install";

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      {isFailed ? (
        <>
          <b>Fleet</b> failed to install certificate{" "}
          <b>{activity.details?.certificate_name}</b> on this host.
          {activity.details?.detail && (
            <> Detail: {activity.details.detail}</>
          )}
        </>
      ) : (
        <>
          <b>Fleet</b> installed certificate{" "}
          <b>{activity.details?.certificate_name}</b> on this host.
        </>
      )}
    </ActivityItem>
  );
};

export default InstalledCertificateActivityItem;
