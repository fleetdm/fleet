import React from "react";

import { IHostCertificate } from "interfaces/certificates";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";

const baseClass = "certificate-details-modal";

interface ICertificateDetailsModalProps {
  certificate: IHostCertificate;
  onExit: () => void;
}

const CertificateDetailsModal = ({
  certificate,
  onExit,
}: ICertificateDetailsModalProps) => {
  return (
    <Modal className={baseClass} title="Certificate details" onExit={onExit}>
      <>
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__section`}>
            <h3>Subject Name</h3>
            <dl>
              <DataSet
                title="Country or region"
                value={certificate.subject.country}
              />
              <DataSet
                title="Organization"
                value={certificate.subject.organization}
              />
              <DataSet
                title="Organizational unit"
                value={certificate.subject.organizational_unit}
              />
              <DataSet
                title="Common name"
                value={certificate.subject.common_name}
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Issuer name</h3>
            <dl>
              <DataSet
                title="Country or region"
                value={certificate.issuer.country}
              />
              <DataSet
                title="Organization"
                value={certificate.issuer.organization}
              />
              <DataSet
                title="Organizational unit"
                value={certificate.issuer.organizational_unit}
              />
              <DataSet
                title="Common name"
                value={certificate.issuer.common_name}
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Validity period</h3>
            <dl>
              <DataSet
                title="Not valid before"
                value={certificate.not_valid_before}
              />
              <DataSet
                title="Not valid after"
                value={certificate.not_valid_after}
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Key info</h3>
            <dl>
              <DataSet title="Algorithm" value={certificate.key_algorithm} />
              <DataSet title="Key size" value={certificate.key_strength} />
              <DataSet title="Key usage" value={certificate.key_usage} />
              <DataSet title="Serial number" value={certificate.serial} />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Basic constraints</h3>
            <dl>
              <DataSet
                title="Certificate authority"
                value={certificate.certificate_authority ? "Yes" : "No"}
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Signature</h3>
            <dl>
              <DataSet
                title="Algorithm"
                value={certificate.signing_algorithm}
              />
            </dl>
          </div>
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default CertificateDetailsModal;
