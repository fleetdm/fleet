import React from "react";

import { ActivityType } from "interfaces/activity";

/** Detail modals for these types hit premium-only endpoints. */
export const PREMIUM_ONLY_DETAIL_ACTIVITIES: ReadonlySet<string> = new Set([
  ActivityType.InstalledSoftware,
  ActivityType.UninstalledSoftware,
  ActivityType.InstalledAppStoreApp,
]);

export const getMdmCommandDisplayName = (
  requestType: string | undefined
): string => {
  if (!requestType) return "";
  const segments = requestType.split("/").filter(Boolean);
  if (segments.length === 0) return requestType;
  const lastSegment = segments[segments.length - 1];
  return segments.length > 1 ? `.../${lastSegment}` : lastSegment;
};

export const formatMdmCommandNameForActivityItem = (
  requestType: string | undefined
) => {
  const displayName = getMdmCommandDisplayName(requestType);
  if (!displayName) {
    return <>a custom MDM command</>;
  }
  return (
    <>
      <b>{displayName}</b> as a custom MDM command
    </>
  );
};
