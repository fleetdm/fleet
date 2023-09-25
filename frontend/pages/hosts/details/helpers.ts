/** Helpers used across the host details and my device pages and components. */

import {
  IHostMdmProfile,
  IWindowsDiskEncryptionStatus,
  MdmProfileStatus,
} from "interfaces/mdm";

const convertWinDiskEncryptionStatusToProfileStatus = (
  diskEncryptionStatus: IWindowsDiskEncryptionStatus
): MdmProfileStatus => {
  return diskEncryptionStatus === "enforcing"
    ? "pending"
    : diskEncryptionStatus;
};

/**
 * Manually generates a profile for the windows disk encryption status. We need
 * this as we don't have a windows disk encryption profile in the `profiles`
 * attribute coming back from the GET /hosts/:id API response.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateWinDiskEncryptionProfile = (
  diskEncryptionStatus: IWindowsDiskEncryptionStatus
): IHostMdmProfile => {
  return {
    profile_id: 0, // This s the only type of profile that can have this number
    name: "Disk Encryption",
    status: convertWinDiskEncryptionStatusToProfileStatus(diskEncryptionStatus),
    detail: "",
    operation_type: null,
  };
};
