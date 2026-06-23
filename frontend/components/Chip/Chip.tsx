import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "chip";

interface IChipProps {
  icon?: IconNames;
  text: string;
  trailingIcon?: IconNames;
  className?: string;
  onClick?: () => void;
  tooltip?: React.ReactNode;
}

const Chip = ({
  icon,
  text,
  trailingIcon,
  className,
  onClick,
  tooltip,
}: IChipProps) => {
  const classNames = classnames(
    baseClass,
    className,
    onClick && `${baseClass}__clickable-chip`
  );

  const content = (
    <>
      {icon && <Icon name={icon} size="small" color="ui-fleet-black-75" />}
      <span className={`${baseClass}__text`}>{text}</span>
      {trailingIcon && (
        <Icon name={trailingIcon} size="small" color="ui-fleet-black-75" />
      )}
    </>
  );

  const chip = onClick ? (
    // use a button element so that the chip can be focused and clicked
    // with the keyboard
    <button type="button" className={classNames} onClick={onClick}>
      {content}
    </button>
  ) : (
    <div className={classNames}>{content}</div>
  );

  if (!tooltip) {
    return chip;
  }

  return (
    <TooltipWrapper
      tipContent={tooltip}
      position="top"
      underline={false}
      showArrow
    >
      {chip}
    </TooltipWrapper>
  );
};

export default Chip;
