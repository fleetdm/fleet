import CustomLink from "components/CustomLink";
import React from "react";

const baseClass = "add-run-script";

interface IAddRunScriptProps {}

const AddRunScript = ({}: IAddRunScriptProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__description-container`}>
        <p className={`${baseClass}__description`}>
          Upload a script to run on hosts that automatically enroll to Fleet.
        </p>
        <CustomLink newTab url="" text="Learn how" />
      </div>
      <span className={`${baseClass}__added-text`}>
        Script will run during setup:
      </span>
    </div>
  );
};

export default AddRunScript;
