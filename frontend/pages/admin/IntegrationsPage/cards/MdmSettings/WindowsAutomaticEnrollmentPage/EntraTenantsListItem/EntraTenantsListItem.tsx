import React from "react";

import ListItem from "components/ListItem";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "entra-tenants-list-item";

interface IEntraTenantsListItemProps {
  tenantId: string;
  onClickDelete: () => void;
}

const EntraTenantsListItem = ({
  tenantId,
  onClickDelete,
}: IEntraTenantsListItemProps) => {
  return (
    <ListItem
      className={baseClass}
      title={tenantId}
      actions={
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              onClick={onClickDelete}
              className={`${baseClass}__action-button`}
              variant="icon"
              ariaLabel={`Delete Microsoft Entra tenant ${tenantId}`}
            >
              <Icon name="trash" />
            </Button>
          )}
        />
      }
    />
  );
};

export default EntraTenantsListItem;
