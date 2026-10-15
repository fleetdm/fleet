import React from "react";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import ListItem from "components/ListItem";

const baseClass = "entra-client-ids-list-item";

interface IEntraClientIDsListItemProps {
  clientId: string;
  onClickDelete: () => void;
}

const EntraClientIDsListItem = ({
  clientId,
  onClickDelete,
}: IEntraClientIDsListItemProps) => {
  return (
    <ListItem
      className={baseClass}
      title={clientId}
      actions={
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              onClick={onClickDelete}
              className={`${baseClass}__action-button`}
              variant="subdued"
              ariaLabel={`Delete Microsoft Entra client ID ${clientId}`}
              icon="trash"
            />
          )}
        />
      }
    />
  );
};

export default EntraClientIDsListItem;
