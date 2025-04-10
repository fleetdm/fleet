import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";
import { IconNames } from "components/icons";

const baseClass = "tag";

interface ITagProps {
  icon: IconNames;
  text: string;
  className?: string;
  onClick?: () => void;
}

const Tag = ({ icon, text, className, onClick }: ITagProps) => {
  const classNames = classnames(
    baseClass,
    className,
    onClick && `${baseClass}__clickable-tag`
  );

  const content = (
    <>
      <Icon name={icon} size="small" color="ui-fleet-black-75" />
      <span className={`${baseClass}__text`}>{text}</span>
    </>
  );

  return onClick ? (
    // use a button element so that the tag can be focused and clicked
    // with the keyboard
    <button className={classNames} onClick={onClick}>
      {content}
    </button>
  ) : (
    <div className={classNames}>{content}</div>
  );
};

export default Tag;
