import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "tag";

interface ITagProps {
  icon?: IconNames;
  text: string;
  trailingIcon?: IconNames;
  className?: string;
  onClick?: () => void;
  /** Optional tooltip shown on hover/focus. Pass a string or JSX. */
  tooltip?: React.ReactNode;
}

const Tag = ({
  icon,
  text,
  trailingIcon,
  className,
  onClick,
  tooltip,
}: ITagProps) => {
  const classNames = classnames(
    baseClass,
    className,
    onClick && `${baseClass}__clickable-tag`
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

  const tag = onClick ? (
    // use a button element so that the tag can be focused and clicked
    // with the keyboard
    <button className={classNames} onClick={onClick}>
      {content}
    </button>
  ) : (
    <div className={classNames}>{content}</div>
  );

  if (!tooltip) {
    return tag;
  }

  return (
    <TooltipWrapper
      tipContent={tooltip}
      position="top"
      underline={false}
      showArrow
    >
      {tag}
    </TooltipWrapper>
  );
};

export default Tag;
