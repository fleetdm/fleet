import React, { useState } from "react";
import { useDebouncedCallback } from "use-debounce";
import { IconNames } from "components/icons";
// @ts-ignore
import InputFieldWithIcon from "../InputFieldWithIcon";

export interface ISearchFieldProps {
  placeholder: string;
  defaultValue?: string;
  onChange: (value: string) => void;
  onClick?: (e: React.MouseEvent) => void;
  icon?: IconNames;
}

const SearchField = ({
  placeholder,
  defaultValue = "",
  onChange,
  onClick,
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
      onChange={onInputChange}
      onClick={onClick}
      iconPosition="start"
      iconSvg={icon}
    />
  );
};

export default SearchField;
