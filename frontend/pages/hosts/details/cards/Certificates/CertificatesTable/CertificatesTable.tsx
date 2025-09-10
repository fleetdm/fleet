import React, { useCallback } from "react";

import { IHostCertificate } from "interfaces/certificates";
import { IGetHostCertificatesResponse } from "services/entities/hosts";
import { IListSort } from "interfaces/list_options";

import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import TableCount from "components/TableContainer/TableCount";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import generateTableConfig from "./CertificatesTableConfig";

const baseClass = "certificates-table";

interface ICertificatesTableProps {
  data: IGetHostCertificatesResponse;
  showHelpText: boolean;
  page: number;
  pageSize: number;
  sortHeader: string;
  sortDirection: string;
  onSelectCertificate: (certificate: IHostCertificate) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onSortChange: ({ order_key, order_direction }: IListSort) => void;
}

const CertificatesTable = ({
  data,
  showHelpText,
  page,
  pageSize,
  sortDirection,
  sortHeader,
  onSelectCertificate,
  onNextPage,
  onPreviousPage,
  onSortChange,
}: ICertificatesTableProps) => {
  const tableConfig = generateTableConfig();

  const onClickTableRow = (row: any) => {
    onSelectCertificate(row.original);
  };

  const onQueryChange = useCallback(
    (newQuery: ITableQueryData) => {
      switch (true) {
        case newQuery.pageIndex > page:
          return onNextPage();
        case newQuery.pageIndex < page:
          return onPreviousPage();
        case newQuery.sortHeader !== sortHeader ||
          newQuery.sortDirection !== sortDirection:
          return onSortChange({
            order_key: newQuery.sortHeader,
            order_direction: newQuery.sortDirection || "asc",
          });
        default:
          return undefined; // noop
      }
    },
    [onNextPage, onPreviousPage, onSortChange, page, sortDirection, sortHeader]
  );

  const helpText = showHelpText ? (
    <p>
      Showing certificates in the system and login (user) keychain. To get all
      certificates, you can query the certificates table.{" "}
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
      pageIndex={page}
      defaultSortHeader={sortHeader}
      defaultSortDirection={sortDirection}
      onQueryChange={onQueryChange}
      disableNextPage={data?.meta.has_next_results === false}
      manualSortBy
    />
  );
};
export default CertificatesTable;
