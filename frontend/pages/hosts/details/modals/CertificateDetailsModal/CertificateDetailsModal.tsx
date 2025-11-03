import React from "react";

import { IHostCertificate } from "interfaces/certificates";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";
import TooltipTruncatedText from "components/TooltipTruncatedText";
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
  // Destructure the certificate object so we can check for presence of values
  const {
    subject: {
      country: subjectCountry,
      organization: subjectOrganization,
      organizational_unit: subjectOrganizationalUnit,
      common_name: subjectCommonName,
    },
    issuer: {
      country: issuerCountry,
      organization: issuerOrganization,
      organizational_unit: issuerOrganizationalUnit,
      common_name: issuerCommonName,
    },
    not_valid_before,
    not_valid_after,
    key_algorithm,
    key_strength,
    key_usage,
    serial,
    certificate_authority,
    signing_algorithm,
  } = certificate;

  let serialDecimal = "";
  try {
    if (serial) {
      // Convert the serial number to decimal and display it if it is less than 2^63 to
      // match keychain and openSSL display behavior
      const serialParsed = BigInt(`0x${serial}`);
      if (serialParsed < BigInt("0x8000000000000000")) {
        serialDecimal = serialParsed.toString(10);
      }
    }
  } catch (e) {
    // The serial couldn't be converted to decimal but this is best effort so not a big deal
    // since we will still show the original representation, whatever it was
  }

  const showSubjectSection = Boolean(
    subjectCountry ||
    subjectOrganization ||
    subjectOrganizationalUnit ||
    subjectCommonName
  );
  const showIssuerNameSection = Boolean(
    issuerCommonName ||
    issuerCountry ||
    issuerOrganization ||
    issuerOrganizationalUnit
  );
  const showValidityPeriodSection = Boolean(
    not_valid_before || not_valid_after
  );
  const showKeyInfoSection = Boolean(
    key_algorithm || key_strength || key_usage || serial
  );
  const showSignatureSection = Boolean(signing_algorithm);

  return (
    <Modal className={baseClass} title="Certificate details" onExit={onExit}>
      <>
        <div className={`${baseClass}__content`}>
          {showSubjectSection && (
            <div className={`${baseClass}__section`}>
              <h3>Subject name</h3>
              <dl>
                {subjectCountry && (
                  <DataSet
                    title="Country or region"
                    value={<TooltipTruncatedText value={subjectCountry} />}
                    orientation="horizontal"
                  />
                )}
                {subjectOrganization && (
                  <DataSet
                    title="Organization"
                    value={
                      <TooltipTruncatedText
                        value={
                          "This is long text to truncate. does it get truncates? it should get truncated. alsjdf;lajsdl;fj;lajsdjasljjjfljk;j"
                        }
                      />
                    }
                    orientation="horizontal"
                  />
                )}
                {subjectOrganizationalUnit && (
                  <DataSet
                    title="Organizational unit"
                    value={
                      <TooltipTruncatedText value={subjectOrganizationalUnit} />
                    }
                    orientation="horizontal"
                  />
                )}
                {subjectCommonName && (
                  <DataSet
                    title="Common name"
                    value={<TooltipTruncatedText value={subjectCommonName} />}
                    orientation="horizontal"
                  />
                )}
              </dl>
            </div>
          )}
          {showIssuerNameSection && (
            <div className={`${baseClass}__section`}>
              <h3>Issuer name</h3>
              <dl>
                {issuerCountry && (
                  <DataSet
                    title="Country or region"
                    value={<TooltipTruncatedText value={issuerCountry} />}
                    orientation="horizontal"
                  />
                )}
                {issuerOrganization && (
                  <DataSet
                    title="Organization"
                    value={<TooltipTruncatedText value={issuerOrganization} />}
                    orientation="horizontal"
                  />
                )}
                {issuerOrganizationalUnit && (
                  <DataSet
                    title="Organizational unit"
                    value={
                      <TooltipTruncatedText value={issuerOrganizationalUnit} />
                    }
                    orientation="horizontal"
                  />
                )}
                {issuerCommonName && (
                  <DataSet
                    title="Common name"
                    value={<TooltipTruncatedText value={issuerCommonName} />}
                    orientation="horizontal"
                  />
                )}
              </dl>
            </div>
          )}
          {showValidityPeriodSection && (
            <div className={`${baseClass}__section`}>
              <h3>Validity period</h3>
              <dl>
                {not_valid_before && (
                  <DataSet
                    title="Not valid before"
                    value={
                      <TooltipTruncatedText
                        value={monthDayYearFormat(not_valid_before)}
                      />
                    }
                    orientation="horizontal"
                  />
                )}
                {not_valid_after && (
                  <DataSet
                    title="Not valid after"
                    value={
                      <TooltipTruncatedText
                        value={monthDayYearFormat(not_valid_after)}
                      />
                    }
                    orientation="horizontal"
                  />
                )}
              </dl>
            </div>
          )}
          {showKeyInfoSection && (
            <div className={`${baseClass}__section`}>
              <h3>Key info</h3>
              <dl>
                {key_algorithm && (
                  <DataSet
                    title="Algorithm"
                    value={<TooltipTruncatedText value={key_algorithm} />}
                    orientation="horizontal"
                  />
                )}
                {!!key_strength && (
                  <DataSet
                    title="Key size"
                    value={<TooltipTruncatedText value={key_strength} />}
                    orientation="horizontal"
                  />
                )}
                {key_usage && (
                  <DataSet
                    title="Key usage"
                    value={<TooltipTruncatedText value={key_usage} />}
                    orientation="horizontal"
                  />
                )}
                {serial && (
                  <DataSet
                    title="Serial number (hex)"
                    value={<TooltipTruncatedText value={serial} />}
                    orientation="horizontal"
                  />
                )}
                {serialDecimal && (
                  <DataSet
                    title="Serial number (decimal)"
                    value={<TooltipTruncatedText value={serialDecimal} />}
                    orientation="horizontal"
                  />
                )}
              </dl>
            </div>
          )}
          {/* will always show this section */}
          <div className={`${baseClass}__section`}>
            <h3>Basic constraints</h3>
            <dl>
              <DataSet
                title="Certificate authority"
                value={
                  <TooltipTruncatedText
                    value={certificate_authority ? "Yes" : "No"}
                  />
                }
                orientation="horizontal"
              />
            </dl>
          </div>
          {showSignatureSection && (
            <div className={`${baseClass}__section`}>
              <h3>Signature</h3>
              <dl>
                <DataSet
                  title="Algorithm"
                  value={<TooltipTruncatedText value={signing_algorithm} />}
                  orientation="horizontal"
                />
              </dl>
            </div>
          )}
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default CertificateDetailsModal;
