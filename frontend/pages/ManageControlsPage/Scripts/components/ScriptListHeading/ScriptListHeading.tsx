import React from "react";

const baseClass = "script-list-heading";

const ScriptListHeading = () => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__heading-group`}>
        <span>Script</span>
      </div>
    </div>
  );
};

export default ScriptListHeading;
