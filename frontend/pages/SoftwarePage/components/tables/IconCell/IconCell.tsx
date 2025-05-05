import React from "react";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import Icon from "components/Icon";
import { COLORS } from "styles/var/colors";
import { IconNames } from "components/icons";

const baseClass = "icon-cell";

interface IIconCellProps {
  iconName: IconNames;
}

const IconCell = ({ iconName }: IIconCellProps) => {
  const tooltipID = uniqueId();

  return (
    <div className={baseClass}>
      <span
        className={`${baseClass}__icon tooltip tooltip__tooltip-icon`}
        data-tip
        data-for={tooltipID}
        data-tip-disable={false}
      >
        <Icon name={iconName} />
      </span>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor={COLORS["tooltip-bg"]}
        id={tooltipID}
        data-html
      >
        <span className={`tooltip__tooltip-text`}>
          {/* TODO: enhance to be dynmaic */}
          Software can be installed on Host details page.
        </span>
      </ReactTooltip>
    </div>
  );
};

export default IconCell;
