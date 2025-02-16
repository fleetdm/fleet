import React from "react";

import { createMockHostCertificate } from "__mocks__/certificatesMock";

import Card from "components/Card";

import CertificatesTable from "./CertificatesTable";

const baseClass = "certificates-card";

interface ICertificatesProps {}

const CertificatesCard = ({}: ICertificatesProps) => {
  return (
    <Card
      className={baseClass}
      borderRadiusSize="xxlarge"
      includeShadow
      paddingSize="xxlarge"
    >
      <h2>Certificates</h2>
      <CertificatesTable data={[createMockHostCertificate()]} />
    </Card>
  );
};

export default CertificatesCard;
