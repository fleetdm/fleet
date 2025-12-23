import React, { useMemo } from "react";
import Select, {
  GroupBase,
  GroupHeadingProps,
  DropdownIndicatorProps,
  components,
} from "react-select-5";
import Icon from "components/Icon";
import CustomDropdownIndicator from "pages/hosts/ManageHostsPage/components/CustomDropdownIndicator";

const baseClass = "fma-filter-select";

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
  helpText?: string;
  isDisabled: boolean;
}

export type FmaStatusGroup = GroupBase<IStatusOption>;

interface IFmaStatusSelectProps {
  value: FmaStatusValue;
  onChange: (value: FmaStatusValue) => void;
  className?: string;
}

/** Simple group heading: just the section title + optional icon */
const FmaGroupHeading = (
  props: GroupHeadingProps<IStatusOption, false, FmaStatusGroup>
) => {
  const { children } = props;

  return (
    <components.GroupHeading {...props}>
      <>
        {children === "macOS" && (
          <Icon name="darwin" className={`${baseClass}__group-heading-icon`} />
        )}
        {children === "Windows" && (
          <Icon name="windows" className={`${baseClass}__group-heading-icon`} />
        )}
        <span className={`${baseClass}__group-heading-text`}>{children}</span>
      </>
    </components.GroupHeading>
  );
};

const ValueContainer = ({ children, ...props }: any) => {
  return (
    components.ValueContainer && (
      <components.ValueContainer {...props}>
        {!!children && <Icon name="filter-alt" className="filter-icon" />}
        {children}
      </components.ValueContainer>
    )
  );
};

const FmaStatusDropdownIndicator = (
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
      label: "All Fleet-maintained apps",
      helpText: "Show apps for all platforms and statuses",
      isDisabled: false,
    };

    const macOptions: IStatusOption[] = [
      {
        value: "macos",
        label: "All macOS apps",
        helpText: "Show all apps for macOS",
        isDisabled: false,
      },
      {
        value: "added_macos",
        label: "Added apps",
        helpText: "Apps already added to this team for macOS",
        isDisabled: false,
      },
      {
        value: "available_macos",
        label: "Available apps",
        helpText: "Apps available to add to this team for macOS",
        isDisabled: false,
      },
    ];

    const windowsOptions: IStatusOption[] = [
      {
        value: "windows",
        label: "All Windows apps",
        helpText: "Show all apps for Windows",
        isDisabled: false,
      },
      {
        value: "added_windows",
        label: "Added apps",
        helpText: "Apps already added to this team for Windows",
        isDisabled: false,
      },
      {
        value: "available_windows",
        label: "Available apps",
        helpText: "Apps available to add to this team for Windows",
        isDisabled: false,
      },
    ];

    return [
      {
        label: undefined,
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

  // Only show helpText in the menu (not for the selected value)
  const formatOptionLabel = (
    option: IStatusOption,
    { context }: { context: "menu" | "value" }
  ) => (
    <div className={`${baseClass}__option`}>
      <div className="dropdown-wrapper-option__label">{option.label}</div>
      {/* {context === "menu" && option.helpText && (
        <div className="dropdown-wrapper-option__help-text">
          {option.helpText}
        </div>
      )} */}
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
          GroupHeading: FmaGroupHeading,
          DropdownIndicator: FmaStatusDropdownIndicator,
          ValueContainer,
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
