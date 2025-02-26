import React from "react";
import { Row } from "react-table";

import { IHostCertificate } from "interfaces/certificates";

import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import TableCount from "components/TableContainer/TableCount";

import generateTableConfig from "./CertificatesTableConfig";

const baseClass = "certificates-table";

interface ICertificatesTableProps {
  data: IHostCertificate[];
  showHelpText: boolean;
  onSelectCertificate: (certificate: IHostCertificate) => void;
}

const CertificatesTable = ({
  data,
  showHelpText,
  onSelectCertificate,
}: ICertificatesTableProps) => {
  const tableConfig = generateTableConfig();

  const onClickTableRow = (row: Row<IHostCertificate>) => {
    onSelectCertificate(row.original);
  };

  const helpText = showHelpText ? (
    <p>
      Showing certificates in the system keychain. To get all certificates, you
      can query the certificates table.{" "}
      <CustomLink
        text="Learn more"
        url="https://fleetdm.com/learn-more-about/certificates-query"
        newTab
      />
    </p>
  ) : null;

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
      renderTableHelpText={() => helpText}
      renderCount={() => <TableCount name="certificates" count={data.length} />}
      disablePagination
    />
  );
};
export default CertificatesTable;
