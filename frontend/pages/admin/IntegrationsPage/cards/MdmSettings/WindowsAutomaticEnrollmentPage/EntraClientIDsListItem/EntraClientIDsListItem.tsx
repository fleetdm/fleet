import React from "react";

import ListItem from "components/ListItem";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

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
              variant="icon"
              ariaLabel={`Delete Microsoft Entra client ID ${clientId}`}
            >
              <Icon name="trash" />
            </Button>
          )}
        />
      }
    />
  );
};

export default EntraClientIDsListItem;
