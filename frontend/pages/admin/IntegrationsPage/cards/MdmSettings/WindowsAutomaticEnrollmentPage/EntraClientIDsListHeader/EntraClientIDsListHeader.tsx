import React from "react";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";

const baseClass = "entra-client-ids-list-header";

interface IEntraClientIDsListHeaderProps {
  onClickAddClientId: () => void;
}

const EntraClientIDsListHeader = ({
  onClickAddClientId,
}: IEntraClientIDsListHeaderProps) => {
  return (
    <div className={baseClass}>
      <span className={`${baseClass}__name`}>Client IDs</span>
      <span className={`${baseClass}__actions`}>
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              variant="secondary"
              className={`${baseClass}__add-button`}
              onClick={onClickAddClientId}
            >
              <>
                <Icon name="plus" />
                Add
              </>
            </Button>
          )}
        />
      </span>
    </div>
  );
};

export default EntraClientIDsListHeader;
