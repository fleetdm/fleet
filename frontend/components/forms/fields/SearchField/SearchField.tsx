import React, { useState } from "react";
import { useDebouncedCallback } from "use-debounce";

import { IconNames } from "components/icons";
import TooltipWrapper from "components/TooltipWrapper";

// @ts-ignore
import InputFieldWithIcon from "../InputFieldWithIcon";

const baseClass = "search-field";

export interface ISearchFieldProps {
  placeholder: string;
  defaultValue?: string;
  onChange: (value: string) => void;
  onClick?: (e: React.MouseEvent) => void;
  clearButton?: boolean;
  icon?: IconNames;
  tooltip?: React.ReactNode;
}

const SearchField = ({
  placeholder,
  defaultValue = "",
  onChange,
  clearButton,
  onClick,
  icon = "search",
  tooltip,
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
    <>
      <TooltipWrapper
        disableTooltip={!tooltip}
        tipContent={tooltip}
        position="top"
        showArrow
        underline={false}
        tooltipClass={`${baseClass}__tooltip-text`}
        className={`${baseClass}__tooltip-container`}
      >
        <InputFieldWithIcon
          name={icon}
          placeholder={placeholder}
          value={searchQueryInput}
          onChange={onInputChange}
          onClick={onClick}
          clearButton={clearButton}
          iconPosition="start"
          iconSvg={icon}
        />
      </TooltipWrapper>
    </>
  );
};

export default SearchField;
