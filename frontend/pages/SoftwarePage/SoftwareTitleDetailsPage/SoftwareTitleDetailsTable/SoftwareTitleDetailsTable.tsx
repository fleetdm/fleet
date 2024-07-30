/** software/titles/:id > Versions section */

import React, { useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import PATHS from "router/paths";

import { ISoftwareTitleVersion } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
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
  teamIdForApi?: number;
  isIPadOSOrIOSApp: boolean;
}

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

const SoftwareTitleDetailsTable = ({
  router,
  data,
  isLoading,
  teamIdForApi,
  isIPadOSOrIOSApp,
}: ISoftwareTitleDetailsTableProps) => {
  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = {
      software_version_id: row.original.id,
    };

    const path = hostsBySoftwareParams
      ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
          hostsBySoftwareParams
        )}`
      : PATHS.MANAGE_HOSTS;

    router.push(path);
  };

  const softwareTableHeaders = useMemo(
    () =>
      generateSoftwareTitleDetailsTableConfig({
        router,
        teamId: teamIdForApi,
        isIPadOSOrIOSApp,
      }),
    [router, teamIdForApi, isIPadOSOrIOSApp]
  );

  const renderVersionsCount = () => (
    <TableCount name="versions" count={data?.length} />
  );

  return (
    <TableContainer
      className={baseClass}
      columnConfigs={softwareTableHeaders}
      data={data}
      isLoading={isLoading}
      emptyComponent={NoVersionsDetected}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      defaultSortHeader={DEFAULT_SORT_HEADER}
      defaultSortDirection={DEFAULT_SORT_DIRECTION}
      disablePagination
      disableMultiRowSelect
      onSelectSingleRow={handleRowSelect}
      renderCount={renderVersionsCount}
    />
  );
};

export default SoftwareTitleDetailsTable;
