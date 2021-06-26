import React from "react";
import classNames from "classnames";

const baseClass = "info-banner";

interface IInfoBannerProps {
  children: React.ReactNode;
  className?: string;
}

const InfoBanner = (props: IInfoBannerProps): JSX.Element => {
  const { children, className } = props;
  const wrapperClasses = classNames(baseClass, className);

  return <div className={wrapperClasses}>{children}</div>;
};

export default InfoBanner;
