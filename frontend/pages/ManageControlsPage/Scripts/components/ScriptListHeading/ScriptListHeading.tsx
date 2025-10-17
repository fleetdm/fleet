import React from "react";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "script-list-heading";

interface IScriptListHeading {
  onClickAddScript: () => void;
}

const ScriptListHeading = ({ onClickAddScript }: IScriptListHeading) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__heading-group`}>
        <span>Script</span>
        <span className={`${baseClass}__button-container`}>
          <Button
            variant="brand-inverse-icon"
            onClick={() => {
              return onClickAddScript();
            }}
          >
            <Icon name="plus" color="core-fleet-green" />
            <span>Add script</span>
          </Button>
        </span>
      </div>
    </div>
  );
};

export default ScriptListHeading;
