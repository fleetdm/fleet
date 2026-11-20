import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "stopped-reporting-microsoft-entra-device-id-activity-item";

const StoppedReportingMicrosoftEntraDeviceIdActivityItem = ({
  activity,
  isSoloActivity,
}: IHostActivityItemComponentProps) => {
  const { entra_device_id, new_entra_device_id, reason } =
    activity.details || {};
  const why =
    reason === "replaced" && new_entra_device_id ? (
      <>
        because the host now reports <b>{new_entra_device_id}</b>
      </>
    ) : (
      "because the host no longer has a trusted certificate for it"
    );

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      isSoloActivity={isSoloActivity}
      hideCancel
      hideShowDetails
    >
      <span>
        <b>Fleet</b> stopped reporting Microsoft Entra device ID{" "}
        <b>{entra_device_id}</b> for this host {why}. Entra keeps the last
        status Fleet reported for it. If this host shouldn&apos;t have reported
        it, disable the device in Entra.
      </span>
    </ActivityItem>
  );
};

export default StoppedReportingMicrosoftEntraDeviceIdActivityItem;
