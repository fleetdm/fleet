import React from "react";
import { ValueContainerProps, components } from "react-select-5";

import Icon from "components/Icon";

const baseClass = "custom-dropdown-indicator";

const CustomValueContainer = ({ children, ...props }: ValueContainerProps) => {
  return (
    <components.ValueContainer {...props}>
      {!!children && (
        <Icon name="filter-alt" className={`${baseClass}__icon`} />
      )}
      {children}
    </components.ValueContainer>
  );
};

export default CustomValueContainer;
