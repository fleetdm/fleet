import React from "react";
import classnames from "classnames";

import FleetIcon from "components/icons/FleetIcon";
import platformIconClass from "utilities/platform_icon_class";

interface IPlatformIconProps {
  className?: string;
  fw?: boolean;
  name: string;
  size?: string;
  title?: string;
}

const baseClass = "platform-icon";

const PlatformIcon = ({
  className,
  name,
  fw,
  size,
  title,
}: IPlatformIconProps): JSX.Element => {
  const iconClasses = classnames(baseClass, className);
  let iconName = platformIconClass(name);

  if (!iconName) {
    iconName = "single-host";
  }

  return (
    <FleetIcon
      className={iconClasses}
      fw={fw}
      name={iconName}
      size={size}
      title={title}
    />
  );
};

export default PlatformIcon;
