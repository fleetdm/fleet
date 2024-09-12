import React from "react";
import classnames from "classnames";

interface IFleetIconProps {
  className?: string;
  fw?: boolean;
  name: string;
  size?: string;
  title?: string;
}

const baseClass = "fleeticon";

const FleetIcon = ({
  className,
  fw,
  name,
  size,
  title,
}: IFleetIconProps): JSX.Element => {
  const iconClasses = classnames(baseClass, `${baseClass}-${name}`, className, {
    [`${baseClass}-fw`]: fw,
    [`${baseClass}-${size}`]: !!size,
  });

  return <i className={iconClasses} title={title} />;
};

export default FleetIcon;
