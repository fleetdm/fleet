import React from "react";
import { ValueContainerProps, components } from "react-select-5";

import Icon from "components/Icon";

const baseClass = "custom-dropdown-indicator";

const CustomDropdownIndicator = ({ props }: ValueContainerProps | any) => {
  const { children } = props;
  // no access to hover state here from react-select so that is done in the scss
  // file of LabelFilterSelect.

  return (
    <components.DropdownIndicator {...props} className={baseClass}>
      <Icon name="filter-alt" className={`${baseClass}__icon`} />
      {children}
    </components.DropdownIndicator>
  );
};

export default CustomDropdownIndicator;
