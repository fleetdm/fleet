import React, { useState } from "react";
import { useDebouncedCallback } from "use-debounce";
import { IconNames } from "components/icons";
// @ts-ignore
import InputFieldWithIcon from "../InputFieldWithIcon";

const baseClass = "search-field";

export interface ISearchFieldProps {
  placeholder: string;
  defaultValue?: string;
  onChange: (value: string) => void;
  icon?: IconNames;
}

const SearchField = ({
  placeholder,
  defaultValue = "",
  onChange,
  icon = "search",
}: ISearchFieldProps): JSX.Element => {
  const [searchQueryInput, setSearchQueryInput] = useState(defaultValue);

  const debouncedOnChange = useDebouncedCallback((newValue: string) => {
    onChange(newValue);
  }, 500);

  const onInputChange = (newValue: string): void => {
    setSearchQueryInput(newValue);
    debouncedOnChange(newValue);
  };

  return (
    <InputFieldWithIcon
      name={icon}
      placeholder={placeholder}
      value={searchQueryInput}
      // inputWrapperClass={`${baseClass}__input-wrapper`}
      onChange={onInputChange}
      iconPosition="start"
      iconSvg={icon}
    />
  );
};

export default SearchField;
