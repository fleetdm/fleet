import React, { useMemo } from "react";
import { IconNames, ICON_MAP } from "components/icons";
import classnames from "classnames";

interface IIconProps {
  name: IconNames;
  color?: "coreVibrantBlue" | "coreFleetBlack";
  direction?: "up" | "down" | "left" | "right";
  className?: string;
}

const baseClass = "icon";

const Icon = ({ name, color, direction, className }: IIconProps) => {
  const classsNames = classnames(baseClass, className);

  // createPassedProps creates a props object that we pass to the specific icon
  // for values that are not null or undefined
  const props = useMemo(() => {
    const createPassedProps = () => {
      return Object.assign({}, { color, direction });
    };

    return createPassedProps();
  }, [color, direction]);

  const IconComponent = ICON_MAP[name];

  return (
    <div className={classsNames}>
      <IconComponent {...props} />
    </div>
  );
};

export default Icon;
