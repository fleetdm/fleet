import React, { useState } from "react";
import { useDebouncedCallback } from "use-debounce/lib";
// @ts-ignore
import InputField from "../InputField";

const baseClass = "search-field";

export interface ISearchFieldProps {
  placeholder: string;
  onChange: (value: string) => void;
}

const SearchField = ({
  placeholder,
  onChange,
}: ISearchFieldProps): JSX.Element => {
  const [searchQueryInput, setSearchQueryInput] = useState("");

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
