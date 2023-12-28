import React, { useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";
import TableContainer from "components/TableContainer";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import { IOSVersionsResponse } from "services/entities/operating_systems";

import generateTableConfig from "pages/DashboardPage/cards/OperatingSystems/OperatingSystemsTableConfig";
import { buildQueryStringFromParams } from "utilities/url";

const baseClass = "software-os-table";

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

interface ISoftwareOSTableProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  data?: IOSVersionsResponse;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  currentPage: number;
  teamId?: number;
  isLoading: boolean;
}

const SoftwareOSTable = ({
  router,
  isSoftwareEnabled,
  data,
  perPage,
  orderDirection,
  orderKey,
  currentPage,
  teamId,
  isLoading,
}: ISoftwareOSTableProps) => {
  const { isSandboxMode, noSandboxHosts } = useContext(AppContext);

  const onQueryChange = useCallback((newTableQuery: ITableQueryData) => {
    console.log("onQueryChange");
  }, []);

  const softwareTableHeaders = useMemo(() => {
    if (!data) return [];
    return generateTableConfig(teamId, {
      includeName: true,
      includeVulnerabilities: true,
    });
  }, [data, teamId]);

  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = {
      software_title_id: row.original.id,
      team_id: teamId,
    };

    const path = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
      hostsBySoftwareParams
    )}`;

    router.push(path);
  };

  const getItemsCountText = () => {
    const count = data?.count;
    if (!data?.os_versions || !count) return "";

    return count === 1 ? `${count} item` : `${count} items`;
  };

  const getLastUpdatedText = () => {
    if (!data?.os_versions || !data?.counts_updated_at) return "";
    return (
      <LastUpdatedText
        lastUpdatedAt={data.counts_updated_at}
        whatToRetrieve={"software"}
      />
    );
  };

  const renderSoftwareCount = () => {
    const itemText = getItemsCountText();
    const lastUpdatedText = getLastUpdatedText();

    if (!itemText) return null;

    return (
      <div className={`${baseClass}__count`}>
        <span>{itemText}</span>
        {lastUpdatedText}
      </div>
    );
  };

  const renderTableFooter = () => {
    return (
      <div>
        Seeing unexpected software or vulnerabilities?{" "}
        <CustomLink
          url={GITHUB_NEW_ISSUE_LINK}
          text="File an issue on GitHub"
          newTab
        />
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={softwareTableHeaders}
        data={data?.os_versions ?? []}
        isLoading={isLoading}
        resultsTitle={"items"}
        emptyComponent={() => (
          <EmptySoftwareTable
            isSoftwareDisabled={!isSoftwareEnabled}
            isSandboxMode={isSandboxMode}
            noSandboxHosts={noSandboxHosts}
          />
        )}
        defaultSortHeader={orderKey}
        defaultSortDirection={orderDirection}
        defaultPageIndex={currentPage}
        manualSortBy
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        // disableNextPage={!data?.meta.has_next_results} TODO: API INTEGRATION: update with new API
        disableNextPage={false}
        searchable={false}
        onQueryChange={onQueryChange}
        stackControls
        renderCount={renderSoftwareCount}
        renderFooter={renderTableFooter}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    </div>
  );
};

export default SoftwareOSTable;
