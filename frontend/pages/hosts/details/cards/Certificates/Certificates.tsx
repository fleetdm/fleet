import React from "react";

import Card from "components/Card";

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
    </Card>
  );
};

export default CertificatesCard;
