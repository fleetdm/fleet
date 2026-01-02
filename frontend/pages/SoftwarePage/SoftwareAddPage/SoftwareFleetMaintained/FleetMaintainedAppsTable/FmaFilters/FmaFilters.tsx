import React, { useMemo } from "react";
import { SingleValue } from "react-select-5";

import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";

const statusBaseClass = "fma-status-select";
const platformBaseClass = "fma-platform-select";

export type FmaPlatformValue = "all" | "macos" | "windows";
export type FmaStatusValue = "all" | "added" | "available";

interface IFmaPlatformFilterProps {
  value: FmaPlatformValue;
  onChange: (value: FmaPlatformValue) => void;
  className?: string;
}

export const FmaPlatformFilter = ({
  value,
  onChange,
  className,
}: IFmaPlatformFilterProps) => {
  const options = useMemo<CustomOptionType[]>(() => {
    return [
      {
        value: "all",
        label: "All platforms",
        isDisabled: false,
      },
      {
        value: "macos",
        label: "macOS",
        isDisabled: false,
      },
      {
        value: "windows",
        label: "Windows",
        isDisabled: false,
      },
    ];
  }, []);

  const handleChange = (newValue: SingleValue<CustomOptionType>) => {
    if (!newValue) return;
    onChange(newValue.value as FmaPlatformValue);
  };

  return (
    <div className={`${platformBaseClass} ${className || ""}`}>
      <DropdownWrapper
        name="fma-platform-filter"
        options={options}
        value={value}
        onChange={handleChange}
        variant="table-filter"
        isSearchable={false}
        placeholder="Filter by platform"
        className={platformBaseClass}
        iconName="filter-alt"
      />
    </div>
  );
};

interface IFmaStatusFilterProps {
  value: FmaStatusValue;
  onChange: (value: FmaStatusValue) => void;
  className?: string;
}

export const FmaStatusFilter = ({
  value,
  onChange,
  className,
}: IFmaStatusFilterProps) => {
  const options = useMemo<CustomOptionType[]>(() => {
    return [
      {
        value: "all",
        label: "All apps",
        helpText: "Show apps regardless of status",
        isDisabled: false,
      },
      {
        value: "added",
        label: "Added apps",
        helpText: "Apps already added to this team",
        isDisabled: false,
      },
      {
        value: "available",
        label: "Available apps",
        helpText: "Apps available to add to this team",
        isDisabled: false,
      },
    ];
  }, []);

  const handleChange = (newValue: SingleValue<CustomOptionType>) => {
    if (!newValue) return;
    onChange(newValue.value as FmaStatusValue);
  };

  return (
    <div className={`${statusBaseClass} ${className || ""}`}>
      <DropdownWrapper
        name="fma-status-filter"
        options={options}
        value={value}
        onChange={handleChange}
        variant="table-filter"
        isSearchable={false}
        placeholder="Filter by status"
        className={statusBaseClass}
        iconName="filter-alt"
      />
    </div>
  );
};
