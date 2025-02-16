import React from "react";
import { Row } from "react-table";

import { IHostCertificate } from "interfaces/certificates";

import TableContainer from "components/TableContainer";

import generateTableConfig from "./CertificatesTableConfig";

const baseClass = "certificates-table";

interface ICertificatesTableProps {
  data: IHostCertificate[];
}

const CertificatesTable = ({ data }: ICertificatesTableProps) => {
  const tableConfig = generateTableConfig();

  const onClickTableRow = (row: Row<IHostCertificate>) => {
    console.log(row.original);
  };

  return (
    <TableContainer<Row<IHostCertificate>>
      className={baseClass}
      columnConfigs={tableConfig}
      data={data}
      emptyComponent={() => null}
      isAllPagesSelected={false}
      showMarkAllPages={false}
      isLoading={false}
      onClickRow={onClickTableRow}
    />
  );
};
export default CertificatesTable;
