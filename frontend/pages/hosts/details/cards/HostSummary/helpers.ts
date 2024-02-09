import { HostDeviceStatus, HostPendingAction } from "interfaces/host";

interface IDeviceStatusTag {
  title: string;
  tagType: "warning" | "error";
  generateTooltip: (platform?: string) => string;
}

// We exclude "unlocked as we dont display any device status for it"
type DeviceStatusTagConfig = Record<
  Exclude<HostDeviceStatus, "unlocked"> | HostPendingAction,
  IDeviceStatusTag
>;

// eslint-disable-next-line import/prefer-default-export
export const DEVICE_STATUS_TAGS: DeviceStatusTagConfig = {
  locked: {
    title: "Locked",
    tagType: "warning",
    generateTooltip: (platform) =>
      platform === "darwin"
        ? "Host is locked. The end user can’t use the host until the six-digit PIN has been entered."
        : "Host is locked. The end user can’t use the host until the host has been unlocked.",
  },
  unlock: {
    title: "Unlock Pending",
    tagType: "warning",
    generateTooltip: () =>
      "Host will unlock when it comes online.  If the host is online, it will unlock the next time it checks in to Fleet.",
  },
  lock: {
    title: "Lock Pending",
    tagType: "warning",
    generateTooltip: () =>
      "Host will lock when it comes online.  If the host is online, it will lock the next time it checks in to Fleet.",
  },
};
