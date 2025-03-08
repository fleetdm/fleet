import React from "react";

import ListItem from "components/ListItem";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { ICertAuthority } from "../../helpers";

const baseClass = "cert-authority-list-item";

interface IActionsProps {
  onEdit: () => void;
  onDelete: () => void;
}

const Actions = ({ onEdit, onDelete }: IActionsProps) => {
  return (
    <>
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            onClick={onEdit}
            className={`${baseClass}__action-button`}
            variant="text-icon"
          >
            <Icon name="pencil" color="ui-fleet-black-75" />
          </Button>
        )}
      />
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            onClick={onDelete}
            className={`${baseClass}__action-button`}
            variant="text-icon"
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        )}
      />
    </>
  );
};

interface ICertAuthorityListItemProps {
  cert: ICertAuthority;
  onClickEdit: () => void;
  onClickDelete: () => void;
}

const CertAuthorityListItem = ({
  cert,
  onClickEdit,
  onClickDelete,
}: ICertAuthorityListItemProps) => {
  return (
    <ListItem
      className={`${baseClass}__list-item`}
      graphic="file-certificate"
      title={cert.name}
      details={cert.name}
      actions={<Actions onEdit={onClickEdit} onDelete={onClickDelete} />}
    />
  );
};

export default CertAuthorityListItem;
