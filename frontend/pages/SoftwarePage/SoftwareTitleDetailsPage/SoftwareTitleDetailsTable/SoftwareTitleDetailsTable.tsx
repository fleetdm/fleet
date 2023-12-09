import React, { useMemo } from "react";
import { InjectedRouter } from "react-router";

import { ISoftwareTitleVersion } from "interfaces/software";

import TableContainer from "components/TableContainer";

import generateSoftwareTitleDetailsTableConfig from "./SoftwareTitleDetailsTableConfig";

const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_SORT_DIRECTION = "desc";

const baseClass = "software-title-details-table";

interface ISoftwareTitleDetailsTableProps {
  router: InjectedRouter;
  data: ISoftwareTitleVersion[];
  isLoading: boolean;
}

const SoftwareTitleDetailsTable = ({
  router,
  data,
  isLoading,
}: ISoftwareTitleDetailsTableProps) => {
  const softwareTableHeaders = useMemo(
    () => generateSoftwareTitleDetailsTableConfig(router),
    [router]
  );

  return (
    <TableContainer
      className={baseClass}
      resultsTitle={data.length === 1 ? "version" : "versions"}
      columns={softwareTableHeaders}
      data={data}
      isLoading={isLoading}
      emptyComponent={() => <p>nothing</p>} // TODO: add empty component
      showMarkAllPages={false}
      isAllPagesSelected={false}
      defaultSortHeader={DEFAULT_SORT_HEADER}
      defaultSortDirection={DEFAULT_SORT_DIRECTION}
      disablePagination
      // TODO: add row click handler
    />
  );
};

export default SoftwareTitleDetailsTable;
