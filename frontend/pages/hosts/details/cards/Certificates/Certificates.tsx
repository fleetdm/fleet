import React from "react";
import classnames from "classnames";

import { IGetHostCertificatesResponse } from "services/entities/hosts";

import { IHostCertificate } from "interfaces/certificates";
import { IListSort } from "interfaces/list_options";
import { HostPlatform } from "interfaces/platform";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import DataError from "components/DataError";

import CertificatesTable from "./CertificatesTable";

const baseClass = "certificates-card";

interface ICertificatesProps {
  data: IGetHostCertificatesResponse;
  hostPlatform: HostPlatform;
  page: number;
  pageSize: number;
  sortHeader: string;
  sortDirection: string;
  isError: boolean;
  isMyDevicePage?: boolean;
  className?: string;
  onSelectCertificate: (certificate: IHostCertificate) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onSortChange: ({ order_key, order_direction }: IListSort) => void;
}

const CertificatesCard = ({
  data,
  hostPlatform,
  isError,
  page,
  pageSize,
  sortHeader,
  sortDirection,
  isMyDevicePage = false,
  className,
  onSelectCertificate,
  onNextPage,
  onPreviousPage,
  onSortChange,
}: ICertificatesProps) => {
  const renderContent = () => {
    if (isError) return <DataError />;

    return (
      <CertificatesTable
        data={data}
        showHelpText={!isMyDevicePage && hostPlatform === "darwin"}
        page={page}
        pageSize={pageSize}
        sortDirection={sortDirection}
        sortHeader={sortHeader}
        onSortChange={onSortChange}
        onSelectCertificate={onSelectCertificate}
        onNextPage={onNextPage}
        onPreviousPage={onPreviousPage}
      />
    );
  };

  const classNames = classnames(baseClass, className);

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <CardHeader header="Certificates" />
      {renderContent()}
    </Card>
  );
};

export default CertificatesCard;
