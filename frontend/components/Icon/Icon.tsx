import React, { useMemo } from "react";
import { IconNames, ICON_MAP } from "components/icons";
import classnames from "classnames";

interface IIconProps {
  name: IconNames;
  color?: "coreVibrantBlue" | "coreFleetBlack";
  direction?: "up" | "down" | "left" | "right";
  className?: string;
  size?: "small" | "medium";
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
    <div className={classNames}>
      <IconComponent {...props} />
    </div>
  );
};

export default Icon;
