import React from "react";
import classNames from "classnames";

const baseClass = "info-banner";

export interface IInfoBannerProps {
  children: React.ReactNode;
  className?: string;
}

const InfoBanner = ({ children, className }: IInfoBannerProps): JSX.Element => {
  const wrapperClasses = classNames(baseClass, className);

  return <div className={wrapperClasses}>{children}</div>;
};

export default InfoBanner;
