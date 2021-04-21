import React from "react";
import Select, { OnChangeHandler, Option } from "react-select";

import classnames from "classnames";
import { IDropdownOption } from "interfaces/dropdownOption";

const baseClass = "autocomplete-dropdown";

interface IAutocompleteDropdown {
  id: string;
  value: string[] | string;
  options: IDropdownOption[];
  placeholder: string;
  isLoading: boolean;
  onChange: OnChangeHandler;
  disabled?: boolean;
  optionComponent?: JSX.Element;
  className?: string;
}

const AutocompleteDropdown = (props: IAutocompleteDropdown) => {
  const {
    className,
    value,
    options,
    isLoading,
    disabled,
    placeholder,
    onChange,
    id,
  } = props;

  const wrapperClass = classnames(baseClass, className);

  return (
    <div className={wrapperClass}>
      <Select
        id={id}
        value={value}
        options={options}
        isLoading={isLoading}
        disabled={disabled}
        placeholder={placeholder}
        onChange={onChange}
        multi
        searchable
      />
    </div>
  );
};

export default AutocompleteDropdown;
