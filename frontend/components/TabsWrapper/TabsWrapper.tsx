import React from "react";

interface ITabsWrapperProps {
  children: React.ReactChild | React.ReactChild[];
}

/*
 * This component exists so we can unify the styles
 * and overwrite the loaded React Tabs styles.
 */
const baseClass = "component__tabs-wrapper";

const TabsWrapper = ({ children }: ITabsWrapperProps) => {
  return <div className={baseClass}>{children}</div>;
};

export default TabsWrapper;
