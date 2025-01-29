import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { DiskEncryptionStatus } from "interfaces/mdm";

const baseClass = "disk-encryption-status-filter";

const DISK_ENCRYPTION_STATUS_OPTIONS: IDropdownOption[] = [
  {
    disabled: false,
    label: "Verified",
    value: "verified",
  },
  {
    disabled: false,
    label: "Verifying",
    value: "verifying",
  },
  {
    disabled: false,
    label: "Action required",
    value: "action_required",
  },
  {
    disabled: false,
    label: "Enforcing",
    value: "enforcing",
  },
  {
    disabled: false,
    label: "Failed",
    value: "failed",
  },
  {
    disabled: false,
    label: "Removing enforcement",
    value: "removing_enforcement",
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
        className={`${baseClass}__status-filter`}
        options={DISK_ENCRYPTION_STATUS_OPTIONS}
        searchable={false}
        onChange={onChange}
      />
    </div>
  );
};

export default DiskEncryptionStatusFilter;
