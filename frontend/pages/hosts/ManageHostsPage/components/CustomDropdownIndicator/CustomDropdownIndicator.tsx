import React from "react";
import { DropdownIndicatorProps, components } from "react-select-5";

import { ILabel } from "interfaces/label";
import Icon from "components/Icon";

import { IEmptyOption, IGroupOption } from "../LabelFilterSelect/helpers";

const baseClass = "custom-dropdown-indicator";

const CustomDropdownIndicator = (
  props: DropdownIndicatorProps<ILabel | IEmptyOption, false, IGroupOption>
) => {
  const { isFocused, selectProps } = props;
  // no access to hover state here from react-select so that is done in the scss
  // file of LabelFilterSelect.
  const color =
    isFocused || selectProps.menuIsOpen
      ? "core-fleet-black"
      : "ui-fleet-black-75";

  return (
    <components.DropdownIndicator {...props} className={baseClass}>
      <Icon
        name="chevron-down"
        color={color}
        className={`${baseClass}__icon`}
      />
    </components.DropdownIndicator>
  );
};

export default CustomDropdownIndicator;
