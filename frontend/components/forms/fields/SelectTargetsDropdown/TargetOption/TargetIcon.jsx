import React from "react";
import classnames from "classnames";

import FleetIcon from "components/icons/FleetIcon";
import targetInterface from "interfaces/target";

const baseClass = "target-option";

const TargetIcon = ({ target }) => {
  const iconName = () => {
    const { name, platform, target_type: targetType } = target;

    if (targetType === "labels") {
      return name === "All Hosts" ? "all-hosts" : "label";
    }

    return platform === "darwin" ? "apple" : platform;
  };

  const { status } = target;

  const targetClasses = classnames(
    `${baseClass}__icon`,
    `${baseClass}__icon--${status}`
  );

  return <FleetIcon name={iconName()} className={targetClasses} />;
};

TargetIcon.propTypes = { target: targetInterface.isRequired };

export default TargetIcon;
