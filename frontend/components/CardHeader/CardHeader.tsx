// To be used within <Card/>
import React from "react";
import classnames from "classnames";

const baseClass = "card-header";

interface ICardHeaderProps {
  header: JSX.Element | string;
  subheader?: JSX.Element | string;
  className?: string;
}

/**
 * A generic CardHeader component that will be used to render content within a CardHeader with a border and
 * and selected background color.
 */
const CardHeader = ({ header, subheader, className }: ICardHeaderProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <h2 className={`${baseClass}__header`}>{header}</h2>
      {subheader && <p className={`${baseClass}__subheader`}>{subheader}</p>}
    </div>
  );
};

export default CardHeader;
