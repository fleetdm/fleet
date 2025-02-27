import React, { useCallback } from "react";

import { IHostCertificate } from "interfaces/certificates";
import { IGetHostCertificatesResponse } from "services/entities/hosts";

import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import TableCount from "components/TableContainer/TableCount";

import generateTableConfig from "./CertificatesTableConfig";
import { ITableQueryData } from "components/TableContainer/TableContainer";

const baseClass = "certificates-table";

interface ICertificatesTableProps {
  data: IGetHostCertificatesResponse;
  showHelpText: boolean;
  page: number;
  pageSize: number;
  onSelectCertificate: (certificate: IHostCertificate) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const CertificatesTable = ({
  data,
  showHelpText,
  page,
  pageSize,
  onSelectCertificate,
  onNextPage,
  onPreviousPage,
}: ICertificatesTableProps) => {
  const tableConfig = generateTableConfig();

  const onClickTableRow = (row: any) => {
    onSelectCertificate(row.original);
  };

  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      console.log(newTableQuery);

      if (page === newTableQuery.pageIndex) return;

      if (newTableQuery.pageIndex > page) {
        onNextPage();
      } else {
        onPreviousPage();
      }
    },
    [onNextPage, onPreviousPage, page]
  );

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
    <TableContainer
      className={baseClass}
      columnConfigs={tableConfig}
      data={data.certificates}
      emptyComponent={() => null}
      isAllPagesSelected={false}
      showMarkAllPages={false}
      isLoading={false}
      disableMultiRowSelect
      onSelectSingleRow={onClickTableRow}
      renderTableHelpText={() => helpText}
      renderCount={() => (
        <TableCount name="certificates" count={data.certificates.length} />
      )}
      pageSize={pageSize}
      defaultPageIndex={page}
      onQueryChange={onQueryChange}
      disableNextPage={data?.meta.has_next_results === false}
    />
  );
};
export default CertificatesTable;
