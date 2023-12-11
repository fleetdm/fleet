import React, { useMemo } from "react";
import { InjectedRouter } from "react-router";

import { ISoftwareTitleVersion } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import generateSoftwareTitleDetailsTableConfig from "./SoftwareTitleDetailsTableConfig";

const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_SORT_DIRECTION = "desc";

const baseClass = "software-title-details-table";

const NoVersionsDetected = (): JSX.Element => {
  return (
    <EmptyTable
      header="No versions detected for this software item."
      info={
        <>
          Expecting to see versions?{" "}
          <CustomLink
            url={GITHUB_NEW_ISSUE_LINK}
            text="File an issue on GitHub"
            newTab
          />
        </>
      }
    />
  );
};

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
      emptyComponent={NoVersionsDetected}
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
