import React from "react";
import { formatDistanceToNow } from "date-fns";

import { ICertificate } from "services/entities/certificates";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import InputField from "components/forms/fields/InputField";

const baseClass = "view-certificate-modal";

interface IViewCertificateModalProps {
  cert: ICertificate;
  onExit: () => void;
}

const ViewCertificateModal = ({ cert, onExit }: IViewCertificateModalProps) => {
  const {
    name,
    certificate_authority_name: caName,
    subject_name: subjectName,
    subject_alternative_name: subjectAlternativeName,
    created_at,
  } = cert;

  return (
    <Modal className={baseClass} title={name} width="large" onExit={onExit}>
      <>
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__summary`}>
            <DataSet title="Certificate authority" value={caName} />
            <DataSet
              title="Added"
              value={`${formatDistanceToNow(new Date(created_at))} ago`}
            />
          </div>
          <InputField
            label="Subject name (SN)"
            name="subjectName"
            type="textarea"
            value={subjectName}
            readOnly
          />
          {subjectAlternativeName && (
            <InputField
              label="Subject alternative name (SAN)"
              name="subjectAlternativeName"
              type="textarea"
              value={subjectAlternativeName}
              readOnly
            />
          )}
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default ViewCertificateModal;
