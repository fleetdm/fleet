import React from "react";

import { IconNames, ICON_MAP } from "components/icons";
import classnames from "classnames";

interface IIconProps {
  name: IconNames;
  className?: string;
}

const baseClass = "icon";

const Icon = ({ name, className }: IIconProps) => {
  const classsNames = classnames(baseClass, className);

  const IconComponent = ICON_MAP[name];
  return (
    <div className={classsNames}>
      <IconComponent />
    </div>
  );
};

export default Icon;
