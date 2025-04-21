import React from "react";

import ListItem from "components/ListItem";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { ICertAuthorityListData } from "../../helpers";

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

const generateCertDetails = (certId: string) => {
  if (certId.includes("ndes")) {
    return "Microsoft Network Device Enrollment Service (NDES)";
  } else if (certId.includes("digicert")) {
    return "DigiCert";
  }

  return "Custom Simple Certificate Enrollment Protocol (SCEP)";
};

interface ICertAuthorityListItemProps {
  cert: ICertAuthorityListData;
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
      className={baseClass}
      graphic="file-certificate"
      title={cert.name}
      details={generateCertDetails(cert.id)}
      actions={<Actions onEdit={onClickEdit} onDelete={onClickDelete} />}
    />
  );
};

export default CertAuthorityListItem;
