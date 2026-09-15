import React from "react";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "host-enrollment-rejected-activity-item";

const HostEnrollmentRejectedActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  let content: React.ReactNode;
  switch (activity.details?.reason) {
    case "one_time_secret_spent":
      content = (
        <>
          An enrollment attempt for this host was rejected because its one-time
          enroll secret was already used. Resend the Fleetd configuration
          profile to issue a new one.
        </>
      );
      break;
    case "one_time_secret_identifier_mismatch":
      content = (
        <>
          An enrollment attempt was rejected because it used this host&apos;s
          one-time enroll secret with a different serial number or hardware
          UUID.
        </>
      );
      break;
    case "shared_secret_for_mdm_managed_host":
      content = (
        <>
          An enrollment attempt for this host was rejected because it used a
          shared enroll secret. Hosts enrolled in Fleet MDM must enroll with
          their one-time enroll secret.
        </>
      );
      break;
    default:
      content = <>An enrollment attempt for this host was rejected.</>;
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

export default HostEnrollmentRejectedActivityItem;
