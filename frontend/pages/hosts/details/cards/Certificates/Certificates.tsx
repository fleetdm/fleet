import React from "react";

import { IHostCertificate } from "interfaces/certificates";
import { HostPlatform } from "interfaces/platform";
import { IGetHostCertificatesResponse } from "services/entities/hosts";

import Card from "components/Card";
import DataError from "components/DataError";

import CertificatesTable from "./CertificatesTable";

const baseClass = "certificates-card";

interface ICertificatesProps {
  data: IGetHostCertificatesResponse;
  hostPlatform: HostPlatform;
  page: number;
  pageSize: number;
  isError: boolean;
  isMyDevicePage?: boolean;
  onSelectCertificate: (certificate: IHostCertificate) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const CertificatesCard = ({
  data,
  hostPlatform,
  isError,
  page,
  pageSize,
  isMyDevicePage = false,
  onSelectCertificate,
  onNextPage,
  onPreviousPage,
}: ICertificatesProps) => {
  const renderContent = () => {
    if (isError) return <DataError />;

    return (
      <CertificatesTable
        data={data}
        showHelpText={!isMyDevicePage && hostPlatform === "darwin"}
        page={page}
        pageSize={pageSize}
        onSelectCertificate={onSelectCertificate}
        onNextPage={onNextPage}
        onPreviousPage={onPreviousPage}
      />
    );
  };

  return (
    <Card
      className={baseClass}
      borderRadiusSize="xxlarge"
      includeShadow
      paddingSize="xxlarge"
    >
      <h2>Certificates</h2>
      {renderContent()}
    </Card>
  );
};

export default CertificatesCard;
