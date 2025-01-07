import React from "react";
import classnames from "classnames";
import { noop } from "lodash";

import Icon from "components/Icon";
import { IconNames } from "components/icons";

const baseClass = "internal-link-cell";

interface IInternalLinkCellProps {
  value: string;
  onClick?: () => void;
  className?: string;
  iconName?: IconNames;
}

/** This cell is used when you want a clickable cell value that does not link
 * to an url. This can be used when you'd like to trigger an action when the
 * cell is clicked such as opening a modal.
 *
 * TODO: can we find a way to combine this with LinkCell. Would we want to do that?
 * Also we can improve naming of this component.
 */
const InternalLinkCell = ({
  value,
  onClick = noop,
  className,
  iconName,
}: IInternalLinkCellProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      {/* The content div is to ensure that the clickable area is contained to
          the text and icon. This is to prevent the entire cell from being
          clickable. TODO: Figure out if this is product wants to hand this.
       */}
      <div className={`${baseClass}__content`} onClick={onClick}>
        <span>{value}</span>
        {iconName && <Icon name={iconName} />}
      </div>
    </div>
  );
};

export default InternalLinkCell;
