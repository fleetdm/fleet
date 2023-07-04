import React, { useState } from "react";
import { useDebouncedCallback } from "use-debounce";
// @ts-ignore
import InputField from "../InputField";

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
    <InputField
      placeholder={placeholder}
      value={searchQueryInput}
      inputWrapperClass={`${baseClass}__input-wrapper`}
      onChange={onInputChange}
    />
  );
};

export default SearchField;
