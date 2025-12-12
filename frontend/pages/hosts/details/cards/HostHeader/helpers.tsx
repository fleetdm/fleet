import React from "react";
import { isMacOS, isIPadOrIPhone } from "interfaces/platform";
import { HostMdmDeviceStatusUIState } from "../../helpers";

interface IDeviceStatusTag {
  title: string;
  tagType: "warning" | "error";
  generateTooltip: (platform: string) => React.ReactNode;
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
    generateTooltip: (platform) => {
      if (isIPadOrIPhone(platform)) {
        return (
          <>
            Host is locked. The end user can&apos;t use the host until
            <br />
            unlocked. To unlock select <b>Actions &gt; Unlock</b>.
          </>
        );
      } else if (isMacOS(platform)) {
        return (
          <>
            Host is locked. The end user can&apos;t use the host until
            <br />
            the six-digit PIN has been entered. To view pin select
            <br />
            <b>Actions &gt; Unlock</b>.
          </>
        );
      }
      return (
        <>
          Host is locked. The end user can&apos;t use the host until
          <br />
          unlocked. To unlock select <b>Actions &gt; Unlock</b>.
        </>
      );
    },
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
    generateTooltip: () => (
      <>
        Host will lock when it comes online. If the host is
        <br />
        online, it will lock the next time it checks in to Fleet.
      </>
    ),
  },
  wiped: {
    title: "WIPED",
    tagType: "error",
    generateTooltip: (platform) =>
      isMacOS(platform)
        ? "Host is wiped. To prevent the host from automatically reenrolling to Fleet, first release the host from Apple Business Manager and then delete the host in Fleet."
        : "Host is wiped.",
  },
  wiping: {
    title: "WIPE PENDING",
    tagType: "error",
    generateTooltip: () =>
      "Host will wipe when it comes online. If the host is online, it will wipe the next time it checks in to Fleet.",
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
  wiping: (
    <>
      You can&apos;t fetch data from <br /> a wiping host.
    </>
  ),
  wiped: (
    <>
      You can&apos;t fetch data from <br /> a wiped host.
    </>
  ),
} as const;
