import React from "react";
import classnames from "classnames";

const baseClass = "card-header";

interface ICardHeaderProps {
  header?: JSX.Element | string;
  subheader?: JSX.Element | string;
  className?: string;
}

/** A generic CardHeader component to be used within Card component
 * that styles header and subheader */
const CardHeader = ({ header, subheader, className }: ICardHeaderProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      {header && <h2 className={`${baseClass}__header`}>{header}</h2>}
      {subheader && <p className={`${baseClass}__subheader`}>{subheader}</p>}
    </div>
  );
};

export default CardHeader;
