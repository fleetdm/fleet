import React, { useMemo, useRef, useState } from "react";
import Select, {
  GroupBase,
  InputActionMeta,
  SelectInstance,
} from "react-select-5";
import classnames from "classnames";
import FormField from "components/forms/FormField";

import CustomDropdownIndicator from "../../../../pages/hosts/ManageHostsPage/components/CustomDropdownIndicator";

// Extending the react-select module to add custom props we need for our custom
// group heading. More info here:
// https://react-select.com/typescript#custom-select-props

// TODO
interface IDropdownOptions {
  name?: string;
  value?: string;
}

interface IDropdownSelectProps {
  className?: string;
  clearable?: boolean;
  searchable?: boolean;
  disabled?: boolean;
  error?: string;
  label?: [] | string;
  labelClassName?: string;
  multi?: boolean;
  name?: string;
  onChange: (object: any) => void; // TODO
  onOpen?: () => void;
  onClose?: () => void;
  options?: any;
  placeholder?: [] | string;
  value: any;
  wrapperClassName?: string;
  parseTarget?: boolean;
  tooltip?: string;
  autoFocus?: boolean;
  hint?: string | any[] | JSX.Element;
}

const baseClass = "dropdown-select";

const DropdownSelect = ({
  className,
  clearable,
  searchable,
  disabled,
  error = "",
  label = "",
  multi,
  name = "targets",
  onChange,
  onOpen,
  onClose,
  options,
  placeholder = "Select one...",
  value,
  wrapperClassName,
  parseTarget,
  tooltip,
  autoFocus,
  hint = "",
}: IDropdownSelectProps) => {
  const selectClasses = classnames(className, `${baseClass}`);

  const handleChange = (selected: any) => {
    // const { multi, onChange, clearable, name, parseTarget } = this.props;

    if (parseTarget) {
      // Returns both name and value
      return onChange({ value: selected.value, name });
    }

    if (clearable && selected === null) {
      return onChange(null);
    }

    if (multi) {
      return onChange(selected.map((obj: any) => obj.value).join(","));
    }

    return onChange(selected.value);
  };

  const formFieldProps = {
    hint,
    label,
    error,
    name,
    tooltip,
  };

  return (
    // <FormField
    //   {...formFieldProps}
    //   type="dropdown"
    //   className={wrapperClassName || "dropdown-select-wrapper"}
    // >
    //   {/* <Select
    //     className={selectClasses}
    //     isClearable={clearable}
    //     isDisabled={disabled}
    //     isMulti={!multi ? false : undefined}
    //     isSearchable={searchable}
    //     name={`${name}-select`}
    //     onChange={handleChange}
    //     options={options}
    //     placeholder={placeholder}
    //     value={value}
    //     autoFocus={autoFocus}
    //     onMenuOpen={onOpen}
    //     onMenuClose={onClose}
    //     components={{
    //       DropdownIndicator: CustomDropdownIndicator,
    //     }}
    //   /> */}
    // </FormField>
    <></>
  );
};

export default DropdownSelect;
