import React from "react";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import ListItem from "components/ListItem";

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
              variant="subdued"
              ariaLabel={`Delete Microsoft Entra tenant ${tenantId}`}
              icon="trash"
            />
          )}
        />
      }
    />
  );
};

export default EntraTenantsListItem;
