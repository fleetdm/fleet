/** Helpers used across the host details and my device pages and components. */
import { HostMdmDeviceStatus, HostMdmPendingAction } from "interfaces/host";
import {
  IHostMdmProfile,
  WindowsDiskEncryptionStatus,
  MdmProfileStatus,
  LinuxDiskEncryptionStatus,
} from "interfaces/mdm";

const convertWinDiskEncryptionStatusToSettingStatus = (
  diskEncryptionStatus: WindowsDiskEncryptionStatus
): MdmProfileStatus => {
  return diskEncryptionStatus === "enforcing"
    ? "pending"
    : diskEncryptionStatus;
};

const generateWindowsDiskEncryptionMessage = (
  status: WindowsDiskEncryptionStatus,
  detail: string
) => {
  if (status === "failed") {
    detail += ". Fleet will retry automatically.";
  }
  return detail;
};

/**
 * Manually generates a setting for the windows disk encryption status. We need
 * this as we don't have a windows disk encryption profile in the `profiles`
 * attribute coming back from the GET /hosts/:id API response.
 */
export const generateWinDiskEncryptionSetting = (
  diskEncryptionStatus: WindowsDiskEncryptionStatus,
  detail: string
): IHostMdmProfile => {
  return {
    profile_uuid: "0", // This s the only type of profile that can have this value
    platform: "windows",
    name: "Disk Encryption",
    status: convertWinDiskEncryptionStatusToSettingStatus(diskEncryptionStatus),
    detail,
    operation_type: null,
  };
};

/**
 * Manually generates a setting for the linux disk encryption status. We need
 * this as we don't have a linux disk encryption setting in the `profiles`
 * attribute coming back from the GET /hosts/:id API response.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateLinuxDiskEncryptionSetting = (
  diskEncryptionStatus: LinuxDiskEncryptionStatus,
  detail: string
): IHostMdmProfile => {
  return {
    profile_uuid: "0", // This s the only type of profile that can have this value
    platform: "linux",
    name: "Disk Encryption",
    status: diskEncryptionStatus,
    detail,
    operation_type: null,
  };
};

export type HostMdmDeviceStatusUIState =
  | "unlocked"
  | "locked"
  | "unlocking"
  | "locking"
  | "wiped"
  | "wiping";

// Exclude the empty string from HostPendingAction as that doesn't represent a
// valid device status.
const API_TO_UI_DEVICE_STATUS_MAP: Record<
  HostMdmDeviceStatus | Exclude<HostMdmPendingAction, "">,
  HostMdmDeviceStatusUIState
> = {
  unlocked: "unlocked",
  locked: "locked",
  unlock: "unlocking",
  lock: "locking",
  wiped: "wiped",
  wipe: "wiping",
};

const deviceUpdatingStates = ["unlocking", "locking", "wiping"] as const;

/**
 * Gets the current UI state for the host device status. This helps us know what
 * to display in the UI depending host device status or pending device actions.
 *
 * This approach was chosen to keep a seperation from the API data and the UI.
 * This seperation helps protect us from changes to the API. It also allows
 * us to calculate which UI state we are in at one place.
 */
export const getHostDeviceStatusUIState = (
  deviceStatus: HostMdmDeviceStatus,
  pendingAction: HostMdmPendingAction
): HostMdmDeviceStatusUIState => {
  if (pendingAction === "") {
    return API_TO_UI_DEVICE_STATUS_MAP[deviceStatus];
  }
  return API_TO_UI_DEVICE_STATUS_MAP[pendingAction];
};

/**
 * Checks if our device status UI state is in an updating state.
 */
export const isDeviceStatusUpdating = (
  deviceStatus: HostMdmDeviceStatusUIState
) => {
  return deviceUpdatingStates.includes(deviceStatus as any);
};
