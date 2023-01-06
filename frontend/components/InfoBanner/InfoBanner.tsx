import React from "react";
import classNames from "classnames";

const baseClass = "info-banner";

export interface IInfoBannerProps {
  children: React.ReactNode;
  className?: string;
  /** Default light purple */
  color?: "yellow";
}

const InfoBanner = ({
  children,
  className,
  color,
}: IInfoBannerProps): JSX.Element => {
  const wrapperClasses = classNames(
    baseClass,
    { [`${baseClass}__${color}`]: !!color },
    className
  );

  return <div className={wrapperClasses}>{children}</div>;
};

export default InfoBanner;
