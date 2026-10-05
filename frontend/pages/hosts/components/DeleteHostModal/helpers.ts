import { IHost } from "interfaces/host";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import {
  HostPlatform,
  isAndroid,
  isIPadOrIPhone,
  isLinuxLike,
  isMacOS,
  isWindows,
} from "interfaces/platform";

export interface IDeleteHostTarget {
  platform: HostPlatform;
  isMdmEnrolledInFleet: boolean;
  mdmEnrollmentStatus: MdmEnrollmentStatus | null;
}

type DeleteCopyGroup =
  | "android"
  | "ios"
  | "macos"
  | "windows"
  | "linux"
  | "other";

/** Platforms whose delete copy reads the same, so a mixed ubuntu/debian
 * selection still gets one message instead of the generic bulk copy. */
const deleteCopyGroup = (platform: string): DeleteCopyGroup => {
  if (isAndroid(platform)) return "android";
  if (isIPadOrIPhone(platform)) return "ios";
  if (isMacOS(platform)) return "macos";
  if (isWindows(platform)) return "windows";
  if (isLinuxLike(platform)) return "linux";
  return "other";
};

/**
 * Returns the platform and MDM state shared by every selected host, so the
 * delete modal can show the per-platform copy. Returns undefined when the
 * selection is mixed, in which case the modal falls back to the generic bulk
 * copy. MDM state only changes the copy for macOS, so it is ignored for other
 * platforms.
 */
export const getSharedDeleteHostTarget = (
  hosts: IHost[]
): IDeleteHostTarget | undefined => {
  if (hosts.length === 0) {
    return undefined;
  }
  const [first, ...rest] = hosts;
  const group = deleteCopyGroup(first.platform);
  if (group === "other") {
    return undefined;
  }
  const target: IDeleteHostTarget = {
    platform: first.platform,
    isMdmEnrolledInFleet: !!first.mdm?.connected_to_fleet,
    mdmEnrollmentStatus: first.mdm?.enrollment_status ?? null,
  };
  const sameCopy = rest.every((host) => {
    if (deleteCopyGroup(host.platform) !== group) {
      return false;
    }
    if (group !== "macos") {
      return true;
    }
    return (
      !!host.mdm?.connected_to_fleet === target.isMdmEnrolledInFleet &&
      (host.mdm?.enrollment_status ?? null) === target.mdmEnrollmentStatus
    );
  });
  return sameCopy ? target : undefined;
};
