import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import React from "react";

const baseClass = "setup-software-process-cell";

interface ISetupSoftwareProcessCell {
  name: string;
}

const SetupSoftwareProcessCell = ({ name }: ISetupSoftwareProcessCell) => {
  return (
    <span className={baseClass}>
      <SoftwareIcon name={name || ""} size="small" />
      <div>
        Install <b>{name || "Unknown software"}</b>
      </div>
    </span>
  );
};

export default SetupSoftwareProcessCell;
