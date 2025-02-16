import React from "react";

import createMockHost from "__mocks__/hostMock";
import { createMockHostCertificate } from "__mocks__/certificatesMock";

import TableContainer from "components/TableContainer";

import generateTableConfig from "./CertificatesTableConfig";

const baseClass = "certificates-table";

interface ICertificatesTableProps {}

const CertificatesTable = ({}: ICertificatesTableProps) => {
  const tableConfig = generateTableConfig();

  return (
    <TableContainer
      className={baseClass}
      columnConfigs={tableConfig}
      data={[createMockHostCertificate()]}
      emptyComponent={() => <>empty</>}
      isAllPagesSelected={false}
      showMarkAllPages={false}
      isLoading={false}
    />
  );
};
export default CertificatesTable;
