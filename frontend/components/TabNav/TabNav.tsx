import React from "react";
import classnames from "classnames";

interface ITabNavProps {
  children: React.ReactNode;
  className?: string;
  secondary?: boolean;
}

/*
 * This component exists so we can unify the styles
 * and overwrite the loaded React Tabs styles.
 */
const baseClass = "tab-nav";

const TabNav = ({
  children,
  className,
  secondary = false,
}: ITabNavProps): JSX.Element => {
  const classNames = classnames(baseClass, className, {
    [`${baseClass}--secondary`]: secondary,
  });

  return <div className={classNames}>{children}</div>;
};

export default TabNav;
