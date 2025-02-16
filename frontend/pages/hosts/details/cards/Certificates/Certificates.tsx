import React from "react";

import { createMockHostCertificate } from "__mocks__/certificatesMock";

import Card from "components/Card";

import CertificatesTable from "./CertificatesTable";
import { IHostCertificate } from "interfaces/certificates";
import { IGetHostCertificatesResponse } from "services/entities/hosts";

const baseClass = "certificates-card";

interface ICertificatesProps {
  data: IGetHostCertificatesResponse;
}

const CertificatesCard = ({ data }: ICertificatesProps) => {
  return (
    <Card
      className={baseClass}
      borderRadiusSize="xxlarge"
      includeShadow
      paddingSize="xxlarge"
    >
      <h2>Certificates</h2>
      <CertificatesTable data={data.certificates} />
    </Card>
  );
};

export default CertificatesCard;
