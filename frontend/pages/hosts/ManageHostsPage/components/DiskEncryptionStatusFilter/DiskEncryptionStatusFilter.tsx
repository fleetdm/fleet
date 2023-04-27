import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { FileVaultProfileStatus } from "interfaces/mdm";

const baseClass = "disk-encryption-status-filter";

const DISK_ENCRYPTION_STATUS_OPTIONS: IDropdownOption[] = [
  {
    disabled: false,
    label: "Verifying",
    value: FileVaultProfileStatus.VERIFYING,
  },
  {
    disabled: false,
    label: "Action required",
    value: FileVaultProfileStatus.ACTION_REQUIRED,
  },
  {
    disabled: false,
    label: "Enforcing",
    value: FileVaultProfileStatus.ENFORCING,
  },
  {
    disabled: false,
    label: "Failed",
    value: FileVaultProfileStatus.FAILED,
  },
  {
    disabled: false,
    label: "Removing enforcement",
    value: FileVaultProfileStatus.REMOVING_ENFORCEMENT,
  },
];

interface IDiskEncryptionStatusFilterProps {
  diskEncryptionStatus: FileVaultProfileStatus;
  onChange: (value: FileVaultProfileStatus) => void;
}

const DiskEncryptionStatusFilter = ({
  diskEncryptionStatus,
  onChange,
}: IDiskEncryptionStatusFilterProps) => {
  const value = diskEncryptionStatus;

  return (
    <div className={baseClass}>
      <Dropdown
        value={value}
        className={`${baseClass}__status_dropdown`}
        options={DISK_ENCRYPTION_STATUS_OPTIONS}
        searchable={false}
        onChange={onChange}
      />
    </div>
  );
};

export default DiskEncryptionStatusFilter;
