import React from "react";
import classnames from "classnames";

interface ITabNavProps {
  children: React.ReactChild | React.ReactChild[];
  className?: string;
  sticky?: boolean;
}

/*
 * This component exists so we can unify the styles
 * and overwrite the loaded React Tabs styles.
 */
const baseClass = "tab-nav";

const TabNav = ({
  children,
  className,
  sticky = false,
}: ITabNavProps): JSX.Element => {
  const classNames = classnames(baseClass, className, {
    [`${baseClass}--sticky`]: sticky,
  });

  return <div className={classNames}>{children}</div>;
};

export default TabNav;
