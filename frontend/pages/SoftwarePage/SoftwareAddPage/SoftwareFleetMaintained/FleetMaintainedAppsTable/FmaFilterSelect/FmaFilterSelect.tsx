// frontend/pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppsTable/FmaFilterSelect/FmaStatusSelect.tsx

import React, { useMemo } from "react";
import Select, {
  GroupBase,
  GroupHeadingProps,
  DropdownIndicatorProps,
} from "react-select-5";

import CustomDropdownIndicator from "pages/hosts/ManageHostsPage/components/CustomDropdownIndicator";
import CustomLabelGroupHeading from "pages/hosts/ManageHostsPage/components/CustomLabelGroupHeading";

const baseClass = "fma-status-select";

export type FmaStatusValue =
  | "all"
  | "macos"
  | "added_macos"
  | "available_macos"
  | "windows"
  | "added_windows"
  | "available_windows";

interface IStatusOption {
  value: FmaStatusValue;
  label: string;
  subtext?: string;
}

export type FmaStatusGroup = GroupBase<IStatusOption>;

interface IFmaStatusSelectProps {
  value: FmaStatusValue;
  onChange: (value: FmaStatusValue) => void;
  className?: string;
}

// Wrapper so types match this select’s generics
const FmaFilterSelectGroupHeading = (
  props: GroupHeadingProps<IStatusOption, false, FmaStatusGroup>
) => {
  return <CustomLabelGroupHeading {...(props as any)} />;
};

const FmaFilterSelectDropdownIndicator = (
  props: DropdownIndicatorProps<IStatusOption, false, FmaStatusGroup>
) => {
  return <CustomDropdownIndicator {...(props as any)} />;
};

const FmaStatusSelect = ({
  value,
  onChange,
  className,
}: IFmaStatusSelectProps) => {
  const options = useMemo<FmaStatusGroup[]>(() => {
    const allOption: IStatusOption = {
      value: "all",
      label: "All apps",
      subtext: "Show apps for all platforms and states",
    };

    const macOptions: IStatusOption[] = [
      {
        value: "macos",
        label: "macOS (any state)",
        subtext: "Apps with any macOS status",
      },
      {
        value: "added_macos",
        label: "macOS added",
        subtext: "Apps already added for macOS",
      },
      {
        value: "available_macos",
        label: "macOS available to add",
        subtext: "Apps that can be added for macOS",
      },
    ];

    const windowsOptions: IStatusOption[] = [
      {
        value: "windows",
        label: "Windows (any state)",
        subtext: "Apps with any Windows status",
      },
      {
        value: "added_windows",
        label: "Windows added",
        subtext: "Apps already added for Windows",
      },
      {
        value: "available_windows",
        label: "Windows available to add",
        subtext: "Apps that can be added for Windows",
      },
    ];

    return [
      {
        label: "General",
        options: [allOption],
      },
      {
        label: "macOS",
        options: macOptions,
      },
      {
        label: "Windows",
        options: windowsOptions,
      },
    ];
  }, []);

  const handleChange = (option: IStatusOption | null) => {
    if (!option) return;
    onChange(option.value);
  };

  const formatOptionLabel = (option: IStatusOption) => (
    <div className={`${baseClass}__option`}>
      <div className={`${baseClass}__option-label`}>{option.label}</div>
      {option.subtext && (
        <div className={`${baseClass}__option-subtext`}>{option.subtext}</div>
      )}
    </div>
  );

  const getValue = () => {
    const allOptions = options.reduce<IStatusOption[]>((acc, group) => {
      return acc.concat(group.options);
    }, []);

    return (
      allOptions.find((opt) => opt.value === value) || options[0].options[0]
    );
  };

  return (
    <div className={`${baseClass} ${className || ""}`}>
      <Select<IStatusOption, false, FmaStatusGroup>
        classNamePrefix={baseClass}
        name="fma-status-select"
        value={getValue()}
        options={options}
        isSearchable={false}
        components={{
          GroupHeading: FmaFilterSelectGroupHeading,
          DropdownIndicator: FmaFilterSelectDropdownIndicator,
        }}
        formatOptionLabel={formatOptionLabel}
        getOptionLabel={(o) => o.label}
        getOptionValue={(o) => o.value}
        onChange={handleChange}
        placeholder="Filter by platform status"
      />
    </div>
  );
};

export default FmaStatusSelect;
