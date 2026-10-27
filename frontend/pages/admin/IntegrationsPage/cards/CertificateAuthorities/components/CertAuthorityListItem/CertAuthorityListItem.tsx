import React from "react";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import ListItem from "components/ListItem";

import RenewDateCell from "../../../MdmSettings/components/RenewDateCell";
import { IRenewDateCellStatusConfig } from "../../../MdmSettings/components/RenewDateCell/RenewDateCell";
import { ICertAuthorityListData } from "../CertificateAuthorityList/CertificateAuthorityList";

const baseClass = "cert-authority-list-item";

const CLIENT_CERT_EXPIRY_STATUS_CONFIG: IRenewDateCellStatusConfig = {
  warning: {
    tooltipText: (
      <>
        The client certificate expires in less than 30 days. <br />
        To renew it, edit the certificate authority and upload a new one.
      </>
    ),
  },
  error: {
    tooltipText: (
      <>
        The client certificate expired. Fleet can&apos;t get certificates from
        EJBCA until you upload a new one.
        <br />
        To renew it, edit the certificate authority and upload a new one.
      </>
    ),
  },
};

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
            variant="subdued"
            icon="pencil"
            ariaLabel="Edit certificate authority"
          />
        )}
      />
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            onClick={onDelete}
            className={`${baseClass}__action-button`}
            variant="subdued"
            icon="trash"
            ariaLabel="Delete certificate authority"
          />
        )}
      />
    </>
  );
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
      details={
        <>
          {cert.description}
          {cert.mtls_cert_expires_at && (
            <span className={`${baseClass}__client-cert-expiry`}>
              Client certificate expires:
              <RenewDateCell
                value={cert.mtls_cert_expires_at}
                statusConfig={CLIENT_CERT_EXPIRY_STATUS_CONFIG}
              />
            </span>
          )}
        </>
      }
      actions={<Actions onEdit={onClickEdit} onDelete={onClickDelete} />}
    />
  );
};

export default CertAuthorityListItem;
