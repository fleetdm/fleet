import classnames from "classnames";
import React, { ReactChild } from "react";

interface ISidePanelContentProps {
  children: ReactChild;
  className?: string;
}

const baseClass = "side-panel-content";

const SidePanelContent = ({
  children,
  className,
}: ISidePanelContentProps): JSX.Element => {
  const classes = classnames(baseClass, className);

  return <div className={classes}>{children}</div>;
};

export default SidePanelContent;
