import React from "react";

import { IHostCertificate } from "interfaces/certificates";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";
import { monthDayYearFormat } from "utilities/date_format";

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
                orientation="horizontal"
              />
              <DataSet
                title="Organization"
                value={certificate.subject.organization}
                orientation="horizontal"
              />
              <DataSet
                title="Organizational unit"
                value={certificate.subject.organizational_unit}
                orientation="horizontal"
              />
              <DataSet
                title="Common name"
                value={certificate.subject.common_name}
                orientation="horizontal"
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Issuer name</h3>
            <dl>
              <DataSet
                title="Country or region"
                value={certificate.issuer.country}
                orientation="horizontal"
              />
              <DataSet
                title="Organization"
                value={certificate.issuer.organization}
                orientation="horizontal"
              />
              <DataSet
                title="Organizational unit"
                value={certificate.issuer.organizational_unit}
                orientation="horizontal"
              />
              <DataSet
                title="Common name"
                value={certificate.issuer.common_name}
                orientation="horizontal"
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Validity period</h3>
            <dl>
              <DataSet
                title="Not valid before"
                value={monthDayYearFormat(certificate.not_valid_before)}
                orientation="horizontal"
              />
              <DataSet
                title="Not valid after"
                value={monthDayYearFormat(certificate.not_valid_after)}
                orientation="horizontal"
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Key info</h3>
            <dl>
              <DataSet
                title="Algorithm"
                value={certificate.key_algorithm}
                orientation="horizontal"
              />
              <DataSet
                title="Key size"
                value={certificate.key_strength}
                orientation="horizontal"
              />
              <DataSet
                title="Key usage"
                value={certificate.key_usage}
                orientation="horizontal"
              />
              <DataSet
                title="Serial number"
                value={certificate.serial}
                orientation="horizontal"
              />
            </dl>
          </div>

          <div className={`${baseClass}__section`}>
            <h3>Basic constraints</h3>
            <dl>
              <DataSet
                title="Certificate authority"
                value={certificate.certificate_authority ? "Yes" : "No"}
                orientation="horizontal"
              />
            </dl>
          </div>
          <div className={`${baseClass}__section`}>
            <h3>Signature</h3>
            <dl>
              <DataSet
                title="Algorithm"
                value={certificate.signing_algorithm}
                orientation="horizontal"
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
