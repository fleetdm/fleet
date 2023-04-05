import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";
import { DiskEncryptionStatus } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

const baseClass = "disk-encryption-status-filter";

const DISK_ENCRYPTION_STATUS_OPTIONS: IDropdownOption[] = [
  {
    disabled: false,
    label: "Applied",
    value: DiskEncryptionStatus.APPLIED,
  },
  {
    disabled: false,
    label: "Action required",
    value: DiskEncryptionStatus.ACTION_REQUIRED,
  },
  {
    disabled: false,
    label: "Enforcing",
    value: DiskEncryptionStatus.ENFORCING,
  },
  {
    disabled: false,
    label: "Failed",
    value: DiskEncryptionStatus.FAILED,
  },
  {
    disabled: false,
    label: "Removing enforcement",
    value: DiskEncryptionStatus.REMOVING_ENFORCEMENT,
  },
];

interface IDiskEncryptionStatusFilterProps {
  diskEncryptionStatus: DiskEncryptionStatus;
  onChange: (value: DiskEncryptionStatus) => void;
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
