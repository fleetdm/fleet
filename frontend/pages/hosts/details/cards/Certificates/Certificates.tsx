import React from "react";

import { IHostCertificate } from "interfaces/certificates";
import { HostPlatform } from "interfaces/platform";
import { IGetHostCertificatesResponse } from "services/entities/hosts";

import Card from "components/Card";

import CertificatesTable from "./CertificatesTable";

const baseClass = "certificates-card";

interface ICertificatesProps {
  data: IGetHostCertificatesResponse;
  hostPlatform: HostPlatform;
  isMyDevicePage?: boolean;
  onSelectCertificate: (certificate: IHostCertificate) => void;
}

const CertificatesCard = ({
  data,
  hostPlatform,
  isMyDevicePage = false,
  onSelectCertificate,
}: ICertificatesProps) => {
  return (
    <Card
      className={baseClass}
      borderRadiusSize="xxlarge"
      includeShadow
      paddingSize="xxlarge"
    >
      <h2>Certificates</h2>
      <CertificatesTable
        data={data.certificates}
        showHelpText={!isMyDevicePage && hostPlatform === "darwin"}
        onSelectCertificate={onSelectCertificate}
      />
    </Card>
  );
};

export default CertificatesCard;
