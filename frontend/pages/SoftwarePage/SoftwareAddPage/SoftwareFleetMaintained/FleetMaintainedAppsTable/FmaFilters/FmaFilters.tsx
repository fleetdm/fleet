import React, { useMemo } from "react";
import { SingleValue } from "react-select-5";

import Slider from "components/forms/fields/Slider/Slider";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";

const statusBaseClass = "fma-status-select";
const platformBaseClass = "fma-platform-select";

export type FmaPlatformValue = "all" | "macos" | "windows";
export type FmaStatusValue = "all" | "available";

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
  const handleChange = () => {
    onChange(value === "all" ? ("available" as FmaStatusValue) : "all");
  };

  const enabled = value === "available";

  return (
    <div className={`${statusBaseClass} ${className || ""}`}>
      <Slider
        onChange={handleChange}
        value={enabled}
        inactiveText="Hide added apps"
        activeText="Hide added apps"
      />
    </div>
  );
};
