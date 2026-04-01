import React from "react";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import { IconNames } from "components/icons";

const baseClass = "icon-cell";

interface IIconCellProps {
  iconName: IconNames;
}

const IconCell = ({ iconName }: IIconCellProps) => {
  return (
    <div className={baseClass}>
      <TooltipWrapper
        tipContent={
          <span className="tooltip__tooltip-text">
            {/* TODO: enhance to be dynmaic */}
            Software can be installed on Host details page.
          </span>
        }
        position="top"
        underline={false}
      >
        <span className={`${baseClass}__icon tooltip tooltip__tooltip-icon`}>
          <Icon name={iconName} />
        </span>
      </TooltipWrapper>
    </div>
  );
};

export default IconCell;
