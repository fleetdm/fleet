import classnames from "classnames";
import React, { ReactChild } from "react";

interface ISidePanelContentProps {
  children: ReactChild;
  className?: string;
}

const baseClass = "side-panel-page";

/**
 * A component that controls the layout and styling of the side panel region of
 * the application.
 */
const SidePanelPage = ({
  children,
  className,
}: ISidePanelContentProps): JSX.Element => {
  const classes = classnames(baseClass, className);

  return <div className={classes}>{children}</div>;
};

export default SidePanelPage;
