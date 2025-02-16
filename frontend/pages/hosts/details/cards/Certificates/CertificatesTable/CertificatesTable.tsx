import React, { useCallback } from "react";
import { Row } from "react-table";

import createMockHost from "__mocks__/hostMock";
import { createMockHostCertificate } from "__mocks__/certificatesMock";
import { IHostCertificate } from "interfaces/certificates";

import TableContainer from "components/TableContainer";

import generateTableConfig from "./CertificatesTableConfig";

const baseClass = "certificates-table";

interface ICertificatesTableProps {}

const CertificatesTable = ({}: ICertificatesTableProps) => {
  const tableConfig = generateTableConfig();

  const onClickTableRow = (row: Row<IHostCertificate>) => {
    console.log(row.original);
  };

  return (
    <TableContainer<Row<IHostCertificate>>
      className={baseClass}
      columnConfigs={tableConfig}
      data={[createMockHostCertificate()]}
      emptyComponent={() => <>empty</>}
      isAllPagesSelected={false}
      showMarkAllPages={false}
      isLoading={false}
      onClickRow={onClickTableRow}
    />
  );
};
export default CertificatesTable;
