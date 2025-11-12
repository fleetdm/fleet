import Graphic from "components/Graphic/Graphic";
import React from "react";

const baseClass = "setup-script-process-cell";

interface ISetupScriptProcessCell {
  name: string;
}

const SetupScriptProcessCell = ({ name }: ISetupScriptProcessCell) => {
  return (
    <span className={baseClass}>
      <Graphic name="file-sh" className={`${baseClass}__icon`} />
      <div>
        Run <b>{name || "Unknown script"}</b>
      </div>
    </span>
  );
};

export default SetupScriptProcessCell;
