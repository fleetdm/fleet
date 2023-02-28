import React, { useMemo } from "react";
import classnames from "classnames";

import { IconNames, ICON_MAP } from "components/icons";
import { Colors } from "styles/var/colors";
import { IconSizes } from "styles/var/icon_sizes";

interface IIconProps {
  name: IconNames;
  color?: Colors;
  direction?: "up" | "down" | "left" | "right";
  className?: string;
  size?: IconSizes;
}

const baseClass = "icon";

const Icon = ({ name, color, direction, className, size }: IIconProps) => {
  const classNames = classnames(baseClass, className);

  // createPassedProps creates a props object that we pass to the specific icon
  // for values that are not null or undefined
  const props = useMemo(() => {
    const createPassedProps = () => {
      return Object.assign(
        {},
        color === undefined ? undefined : { color },
        direction === undefined ? undefined : { direction },
        size === undefined ? undefined : { size }
      );
    };

    return createPassedProps();
  }, [color, direction, size]);

  const IconComponent = ICON_MAP[name];

  return (
    <div className={classNames} data-testid="icon">
      <IconComponent {...props} />
    </div>
  );
};

export default Icon;
