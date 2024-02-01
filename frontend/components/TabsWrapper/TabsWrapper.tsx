import React from "react";
import classnames from "classnames";

interface ITabsWrapperProps {
  children: React.ReactChild | React.ReactChild[];
  className?: string;
}

/*
 * This component exists so we can unify the styles
 * and overwrite the loaded React Tabs styles.
 */
const baseClass = "component__tabs-wrapper";

const TabsWrapper = ({
  children,
  className,
}: ITabsWrapperProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  return <div className={classNames}>{children}</div>;
};

export default TabsWrapper;
