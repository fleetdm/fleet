import React from "react";
import classnames from "classnames";

interface ITabTextProps {
  className?: string;
  children: React.ReactNode;
}

/*
 * This component exists so we can unify the styles
 * and add styles to react-tab text.
 */
const baseClass = "tab-text";

const TabText = ({ className, children }: ITabTextProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  return <div className={classNames}>{children}</div>;
};

export default TabText;
