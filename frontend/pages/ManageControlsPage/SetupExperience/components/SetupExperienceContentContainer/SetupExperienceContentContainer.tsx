import React from "react";
import classnames from "classnames";

const baseClass = "setup-experience-content-container";

interface ISetupExperienceContentContainerProps {
  children: React.ReactNode;
  className?: string;
}

const SetupExperienceContentContainer = ({
  children,
  className,
}: ISetupExperienceContentContainerProps) => {
  const classNames = classnames(baseClass, className);
  return <div className={classNames}>{children}</div>;
};

export default SetupExperienceContentContainer;
