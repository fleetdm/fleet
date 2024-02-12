import React from "react";
import { HostMdmDeviceStatusUIState } from "../../helpers";

interface IDeviceStatusTag {
  title: string;
  tagType: "warning" | "error";
  generateTooltip: (platform?: string) => string;
}

type HostMdmDeviceStatusUIStateNoUnlock = Exclude<
  HostMdmDeviceStatusUIState,
  "unlocked"
>;

// We exclude "unlocked" as we dont display any device status tag for it
type DeviceStatusTagConfig = Record<
  HostMdmDeviceStatusUIStateNoUnlock,
  IDeviceStatusTag
>;

export const DEVICE_STATUS_TAGS: DeviceStatusTagConfig = {
  locked: {
    title: "LOCKED",
    tagType: "warning",
    generateTooltip: (platform) =>
      platform === "darwin"
        ? "Host is locked. The end user can’t use the host until the six-digit PIN has been entered."
        : "Host is locked. The end user can’t use the host until the host has been unlocked.",
  },
  unlocking: {
    title: "UNLOCK PENDING",
    tagType: "warning",
    generateTooltip: () =>
      "Host will unlock when it comes online.  If the host is online, it will unlock the next time it checks in to Fleet.",
  },
  locking: {
    title: "LOCK PENDING",
    tagType: "warning",
    generateTooltip: () =>
      "Host will lock when it comes online.  If the host is online, it will lock the next time it checks in to Fleet.",
  },
};

// We exclude "unlocked" as we dont display a tooltip for it.
export const REFETCH_TOOLTIP_MESSAGES: Record<
  HostMdmDeviceStatusUIStateNoUnlock | "offline",
  JSX.Element
> = {
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
