import React from "react";
import classnames from "classnames";

import { ILabel } from "interfaces/label";
import { PLATFORM_LABEL_DISPLAY_NAMES } from "utilities/constants";
import darwinIcon from "../../../../../assets/images/icon-darwin-fleet-black-16x16@2x.png";
import linuxIcon from "../../../../../assets/images/icon-linux-fleet-black-16x16@2x.png";
import ubuntuIcon from "../../../../../assets/images/icon-ubuntu-fleet-black-16x16@2x.png";
import centosIcon from "../../../../../assets/images/icon-centos-fleet-black-16x16@2x.png";
import windowsIcon from "../../../../../assets/images/icon-windows-fleet-black-16x16@2x.png";

const baseClass = "panel-group-item";

const displayName = (name: string) => {
  return PLATFORM_LABEL_DISPLAY_NAMES[name] || name;
};

const displayIcon = (name: string) => {
  switch (name) {
    case "macOS":
      return <img src={darwinIcon} alt="Apple icon" />;
    case "Linux":
      return <img src={linuxIcon} alt="Linux icon" />;
    case "Ubuntu Linux":
      return <img src={ubuntuIcon} alt="Ubuntu icon" />;
    case "CentOS Linux":
      return <img src={centosIcon} alt="Centos icon" />;
    case "Windows":
      return <img src={windowsIcon} alt="Windows icon" />;
    default:
      return null;
  }
};

interface IPanelGroupItemProps {
  item: ILabel;
  onLabelClick: () => void;
  isSelected: boolean;
}

const PanelGroupItem = ({
  item,
  onLabelClick,
  isSelected,
}: IPanelGroupItemProps): JSX.Element => {
  const {
    count,
    display_text: displayText,
    label_type: labelType,
    name,
  } = item;

  const wrapperClassName = classnames(
    baseClass,
    "button",
    "button--contextual-nav-item",
    `${baseClass}__${displayText.toLowerCase().replace(" ", "-")}`,
    {
      [`${baseClass}--selected`]: isSelected,
    }
  );

  return (
    <button className={wrapperClassName} onClick={onLabelClick}>
      <div className={`${baseClass}__flexy`}>
        <span className={`${baseClass}__name`}>
          {labelType === "builtin" && displayIcon(displayName(name))}
          &nbsp;
          {displayName(name)}
        </span>
        <span className={`${baseClass}__count`}>{count}</span>
      </div>
    </button>
  );
};

export default PanelGroupItem;
