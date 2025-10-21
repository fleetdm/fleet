import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
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
      <span className={`${baseClass}__heading-title`}>Scripts</span>
      <span className={`${baseClass}__heading-actions`}>
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              variant="brand-inverse-icon"
              className={`${baseClass}__add-button`}
              onClick={onClickAddScript}
              iconStroke
            >
              <>
                <Icon name="plus" color="core-fleet-green" />
                Add script
              </>
            </Button>
          )}
        />
      </span>
    </div>
  );
};

export default ScriptListHeading;
