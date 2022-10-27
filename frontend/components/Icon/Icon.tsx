import React, { useMemo } from "react";
import { IconNames, ICON_MAP } from "components/icons";
import classnames from "classnames";

interface IIconProps {
  name: IconNames;
  color?: string;
  className?: string;
}

const baseClass = "icon";

const Icon = ({ name, color, className }: IIconProps) => {
  const classsNames = classnames(baseClass, className);

  // createPassedProps creates a props object that we pass to the specific icon
  // for values that are not null or undefined
  const props = useMemo(() => {
    const createPassedProps = () => {
      return Object.assign({}, color === undefined ? undefined : { color });
    };

    return createPassedProps();
  }, [color]);

  const IconComponent = ICON_MAP[name];

  return (
    <div className={classsNames}>
      <IconComponent {...props} />
    </div>
  );
};

export default Icon;
