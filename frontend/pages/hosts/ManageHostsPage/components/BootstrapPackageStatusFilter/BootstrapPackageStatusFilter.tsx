import React from "react";

import { IDropdownOption } from "interfaces/dropdownOption";
import { BootstrapPackageStatus } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

const baseClass = "bootstrap-package-status-filter";

const BOOTSTRAP_PACKAGE_STATUS: IDropdownOption[] = [
  {
    disabled: false,
    label: "Applied",
    value: BootstrapPackageStatus.APPLIED,
  },
  {
    disabled: false,
    label: "Pending",
    value: BootstrapPackageStatus.PENDING,
  },
  {
    disabled: false,
    label: "Failed",
    value: BootstrapPackageStatus.FAILED,
  },
];

interface IBootstrapPackageStatusFilterProps {
  bootstrapPackageStatus: BootstrapPackageStatus;
  onChange: (value: BootstrapPackageStatus) => void;
}

const BootstrapPackageStatusFilter = ({
  bootstrapPackageStatus,
  onChange,
}: IBootstrapPackageStatusFilterProps) => {
  const value = bootstrapPackageStatus;

  return (
    <div className={baseClass}>
      <Dropdown
        value={value}
        className={`${baseClass}__status_dropdown`}
        options={BOOTSTRAP_PACKAGE_STATUS}
        searchable={false}
        onChange={onChange}
      />
    </div>
  );
};

export default BootstrapPackageStatusFilter;
