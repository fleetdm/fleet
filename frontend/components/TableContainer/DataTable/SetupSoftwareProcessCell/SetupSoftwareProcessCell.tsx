import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import React from "react";

const baseClass = "setup-software-process-cell";

interface ISetupSoftwareProcessCell {
  name: string;
  url?: string | null;
}

const SetupSoftwareProcessCell = ({ name, url }: ISetupSoftwareProcessCell) => {
  return (
    <span className={baseClass}>
      <SoftwareIcon name={name || ""} size="small" url={url} />
      <div>
        Install <b>{name || "Unknown software"}</b>
      </div>
    </span>
  );
};

export default SetupSoftwareProcessCell;
