import React from "react";

const baseClass = "script-list-heading";

const ScriptListHeading = () => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__heading-group`}>
        <span>Script</span>
      </div>
      <div
        className={`${baseClass}__heading-group ${baseClass}__actions-heading`}
      >
        <span>Actions</span>
      </div>
    </div>
  );
};

export default ScriptListHeading;
