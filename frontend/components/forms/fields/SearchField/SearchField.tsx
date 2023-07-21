import React, { useState } from "react";
import { useDebouncedCallback } from "use-debounce";
// @ts-ignore
import InputFieldWithIcon from "../InputFieldWithIcon";

const baseClass = "search-field";

export interface ISearchFieldProps {
  placeholder: string;
  defaultValue?: string;
  onChange: (value: string) => void;
}

const SearchField = ({
  placeholder,
  defaultValue = "",
  onChange,
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
      name="search"
      placeholder={placeholder}
      value={searchQueryInput}
      // inputWrapperClass={`${baseClass}__input-wrapper`}
      onChange={onInputChange}
      iconPosition="start"
      iconSvg="search"
    />
  );
};

export default SearchField;
