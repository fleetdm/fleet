import React from "react";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";

const baseClass = "entra-tenants-list-header";

interface IEntraTenantsListHeaderProps {
  onClickAddTenant: () => void;
}

const EntraTenantsListHeader = ({
  onClickAddTenant,
}: IEntraTenantsListHeaderProps) => {
  return (
    <div className={baseClass}>
      <span className={`${baseClass}__name`}>Tenant ID</span>
      <span className={`${baseClass}__actions`}>
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              variant="inverse"
              className={`${baseClass}__add-button`}
              onClick={onClickAddTenant}
              iconStroke
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

export default EntraTenantsListHeader;
