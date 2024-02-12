import React from "react";
import { HostMdmDeviceStatusUIState } from "../../helpers";

interface IDeviceStatusTag {
  title: string;
  tagType: "warning" | "error";
  generateTooltip: (platform?: string) => string;
}

// We exclude "unlocked as we dont display any device status for it"
type DeviceStatusTagConfig = Record<
  Exclude<HostMdmDeviceStatusUIState, "unlocked">,
  IDeviceStatusTag
>;

export const DEVICE_STATUS_TAGS: DeviceStatusTagConfig = {
  locked: {
    title: "Locked",
    tagType: "warning",
    generateTooltip: (platform) =>
      platform === "darwin"
        ? "Host is locked. The end user can’t use the host until the six-digit PIN has been entered."
        : "Host is locked. The end user can’t use the host until the host has been unlocked.",
  },
  unlocking: {
    title: "Unlock Pending",
    tagType: "warning",
    generateTooltip: () =>
      "Host will unlock when it comes online.  If the host is online, it will unlock the next time it checks in to Fleet.",
  },
  locking: {
    title: "Lock Pending",
    tagType: "warning",
    generateTooltip: () =>
      "Host will lock when it comes online.  If the host is online, it will lock the next time it checks in to Fleet.",
  },
};

export const REFETCH_TOOLTIP_MESSAGES = {
  offline: (
    <>
      You can&apos;t fetch data from <br /> an offline host.
    </>
  ),
  unlocking: (
    <>
      You can&apos;t fetch data from <br /> an unlocking host.
    </>
  ),
  locking: (
    <>
      You can&apos;t fetch data from <br /> a locking host.
    </>
  ),
  locked: (
    <>
      You can&apos;t fetch data from <br /> a locked host.
    </>
  ),
} as const;
