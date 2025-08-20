/** software/titles/:id > Versions section */

import React, { useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import PATHS from "router/paths";

import { ISoftwareTitleVersion } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";
import Card from "components/Card";

import generateSoftwareTitleVersionsTableConfig from "./TitleVersionsTableConfig";

const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_PAGE_SIZE = 10;

const baseClass = "software-title-versions-table";

const TitleVersionsLastUpdatedInfo = (lastUpdatedAt: string) => {
  return (
    <LastUpdatedText
      lastUpdatedAt={lastUpdatedAt}
      customTooltipText={
        <>
          The last time software data was <br />
          updated, including vulnerabilities <br />
          and host counts.
        </>
      }
    />
  );
};

const NoVersionsDetected = (isAvailableForInstall = false): JSX.Element => {
  return (
    <Card borderRadiusSize="medium">
      <EmptyTable
        header={
          isAvailableForInstall
            ? "No versions detected."
            : "No versions detected for this software item."
        }
        info={
          isAvailableForInstall ? (
            "Install this software on a host to see versions."
          ) : (
            <>
              Expecting to see versions?{" "}
              <CustomLink
                url={GITHUB_NEW_ISSUE_LINK}
                text="File an issue on GitHub"
                newTab
              />
            </>
          )
        }
      />
    </Card>
  );
};

interface ITitleVersionsTableProps {
  router: InjectedRouter;
  data: ISoftwareTitleVersion[];
  isLoading: boolean;
  teamIdForApi?: number;
  isIPadOSOrIOSApp: boolean;
  isAvailableForInstall?: boolean;
  countsUpdatedAt?: string;
}

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

const TitleVersionsTable = ({
  router,
  data,
  isLoading,
  teamIdForApi,
  isIPadOSOrIOSApp,
  isAvailableForInstall,
  countsUpdatedAt,
}: ITitleVersionsTableProps) => {
  const handleRowSelect = (row: IRowProps) => {
    if (row.original.id) {
      const softwareVersionId = row.original.id;

      const softwareVersionDetailsPath = getPathWithQueryParams(
        PATHS.SOFTWARE_VERSION_DETAILS(softwareVersionId.toString()),
        { team_id: teamIdForApi }
      );

      router.push(softwareVersionDetailsPath);
    }
  };

  const softwareTableHeaders = useMemo(
    () =>
      generateSoftwareTitleVersionsTableConfig({
        teamId: teamIdForApi,
        isIPadOSOrIOSApp,
      }),
    [teamIdForApi, isIPadOSOrIOSApp]
  );

  const renderVersionsCount = () => (
    <>
      {data?.length > 0 && <TableCount name="versions" count={data?.length} />}
      {countsUpdatedAt && TitleVersionsLastUpdatedInfo(countsUpdatedAt)}
    </>
  );

  return (
    <TableContainer
      className={baseClass}
      columnConfigs={softwareTableHeaders}
      data={data}
      isLoading={isLoading}
      emptyComponent={() => NoVersionsDetected(isAvailableForInstall)}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      defaultSortHeader={DEFAULT_SORT_HEADER}
      defaultSortDirection={DEFAULT_SORT_DIRECTION}
      pageSize={DEFAULT_PAGE_SIZE}
      isClientSidePagination
      disableMultiRowSelect
      onSelectSingleRow={handleRowSelect}
      renderCount={renderVersionsCount}
      hideFooter={data?.length <= DEFAULT_PAGE_SIZE} // Removes footer space
    />
  );
};

export default TitleVersionsTable;
