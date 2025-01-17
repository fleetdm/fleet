/** software/os OS tab > Table */

import React, { useCallback, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import PATHS from "router/paths";

import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";
import TableContainer from "components/TableContainer";
import LastUpdatedText from "components/LastUpdatedText";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import { IOSVersionsResponse } from "services/entities/operating_systems";

import generateTableConfig from "pages/DashboardPage/cards/OperatingSystems/OSTableConfig";
import { buildQueryStringFromParams } from "utilities/url";
import { getNextLocationPath } from "utilities/helpers";
import { SelectedPlatform } from "interfaces/platform";

const baseClass = "software-os-table";

interface IRowProps extends Row {
  original: {
    os_version_id?: string;
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
  resetPageIndex: boolean;
  platform?: SelectedPlatform;
}

const PLATFORM_FILTER_OPTIONS = [
  {
    disabled: false,
    label: "All platforms",
    value: "all",
  },
  {
    disabled: false,
    label: "macOS",
    value: "darwin",
  },
  {
    disabled: false,
    label: "Windows",
    value: "windows",
  },
  {
    disabled: false,
    label: "Linux",
    value: "linux",
  },
  {
    disabled: false,
    label: "ChromeOS",
    value: "chrome",
  },
  {
    disabled: false,
    label: "iOS",
    value: "ios",
  },
  {
    disabled: false,
    label: "iPadOS",
    value: "ipados",
  },
];

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
  resetPageIndex,
  platform,
}: ISoftwareOSTableProps) => {
  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "sortDirection":
            return val !== orderDirection;
          case "sortHeader":
            return val !== orderKey;
          case "pageIndex":
            return val !== currentPage;
          case "platform":
            return val !== platform;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [platform, currentPage, orderDirection, orderKey]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      return {
        team_id: teamId,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };
    },
    [teamId]
  );

  const onQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the initial render.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_OS,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, generateNewQueryParams, router]
  );

  const softwareTableHeaders = useMemo(() => {
    if (!data) return [];
    return generateTableConfig(teamId, router, {
      includeName: true,
      includeVulnerabilities: true,
      includeIcon: true,
    });
  }, [data, router, teamId]);

  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = {
      os_version_id: row.original.os_version_id,
      team_id: teamId,
    };

    const path = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
      hostsBySoftwareParams
    )}`;

    router.push(path);
  };

  // Determines if a user should be able to filter the table
  const hasData = data?.os_versions && data?.os_versions.length > 0;
  const hasPlatformFilter = platform !== "all";

  const showFilterHeaders = isSoftwareEnabled && (hasData || hasPlatformFilter);

  const renderSoftwareCount = () => {
    if (!data) return null;

    return (
      <>
        <TableCount name="items" count={data?.count} />
        {showFilterHeaders && data?.counts_updated_at && (
          <LastUpdatedText
            lastUpdatedAt={data.counts_updated_at}
            customTooltipText={
              <>
                The last time software data was <br />
                updated, including vulnerabilities <br />
                and host counts.
              </>
            }
          />
        )}
      </>
    );
  };

  const renderTableHelpText = () => {
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

  const handlePlatformFilterDropdownChange = (
    platformSelected: SingleValue<CustomOptionType>
  ) => {
    router?.replace(
      getNextLocationPath({
        pathPrefix: PATHS.SOFTWARE_OS,
        queryParams: {
          team_id: teamId,
          order_direction: orderDirection,
          order_key: orderKey,
          page: 0,
          platform: platformSelected?.value,
        },
      })
    );
  };

  const renderPlatformDropdown = () => {
    return (
      <DropdownWrapper
        name="os-platform-dropdown"
        value={platform || "all"}
        className={`${baseClass}__platform-dropdown`}
        options={PLATFORM_FILTER_OPTIONS}
        onChange={handlePlatformFilterDropdownChange}
        tableFilter
      />
    );
  };

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={softwareTableHeaders}
        data={data?.os_versions ?? []}
        isLoading={isLoading}
        resultsTitle="items"
        emptyComponent={() => (
          <EmptySoftwareTable
            tableName="operating systems"
            isSoftwareDisabled={!isSoftwareEnabled}
            noSearchQuery // non-searchable table renders not detecting by default
          />
        )}
        defaultSortHeader={orderKey}
        defaultSortDirection={orderDirection}
        defaultPageIndex={currentPage}
        manualSortBy
        pageSize={perPage}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        customControl={showFilterHeaders ? renderPlatformDropdown : undefined}
        disableNextPage={!data?.meta.has_next_results}
        searchable={false}
        onQueryChange={onQueryChange}
        renderCount={renderSoftwareCount}
        renderTableHelpText={renderTableHelpText}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
        resetPageIndex={resetPageIndex}
      />
    </div>
  );
};

export default SoftwareOSTable;
