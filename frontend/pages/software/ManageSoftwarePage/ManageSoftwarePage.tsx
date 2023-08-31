import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { Row } from "react-table";
import PATHS from "router/paths";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import { RouteProps } from "react-router/lib/Route";
import { isEmpty, isEqual } from "lodash";
// import { useDebouncedCallback } from "use-debounce";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import {
  IConfig,
  CONFIG_DEFAULT_RECENT_VULNERABILITY_MAX_AGE_IN_DAYS,
} from "interfaces/config";
import {
  IJiraIntegration,
  IZendeskIntegration,
  IIntegrations,
} from "interfaces/integration";
import { ISoftwareResponse, ISoftwareCountResponse } from "interfaces/software";
import { ITeamConfig } from "interfaces/team";
import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";

import configAPI from "services/entities/config";
import softwareAPI, {
  ISoftwareCountQueryKey,
  ISoftwareQueryKey,
} from "services/entities/software";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import {
  GITHUB_NEW_ISSUE_LINK,
  VULNERABLE_DROPDOWN_OPTIONS,
} from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import TableDataError from "components/DataError";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import LastUpdatedText from "components/LastUpdatedText";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TeamsDropdown from "components/TeamsDropdown";
import { getNextLocationPath } from "utilities/helpers";

import EmptySoftwareTable from "../components/EmptySoftwareTable";

import generateSoftwareTableHeaders from "./SoftwareTableConfig";
import ManageAutomationsModal from "./components/ManageAutomationsModal";

interface IManageSoftwarePageProps {
  route: RouteProps;
  router: InjectedRouter;
  location: {
    pathname: string;
    query: {
      team_id?: string;
      vulnerable?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
    };
    search: string;
  };
}

interface ISoftwareConfigQueryKey {
  scope: string;
  teamId?: number;
}

interface ISoftwareAutomations {
  webhook_settings: {
    vulnerabilities_webhook: IWebhookSoftwareVulnerabilities;
  };
  integrations: {
    jira: IJiraIntegration[];
    zendesk: IZendeskIntegration[];
  };
}

interface IRowProps extends Row {
  original: {
    id?: number;
  };
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_PAGE_SIZE = 20;

const baseClass = "manage-software-page";

const ManageSoftwarePage = ({
  route,
  router,
  location,
}: IManageSoftwarePageProps): JSX.Element => {
  const routeTemplate = route?.path ?? "";
  const queryParams = location.query;
  const {
    config: globalConfig,
    isFreeTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isPremiumTier,
    isSandboxMode,
    noSandboxHosts,
    filteredSoftwarePath,
    setFilteredSoftwarePath,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const {
    currentTeamId,
    isAnyTeamSelected,
    isRouteOk,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const canManageAutomations =
    isGlobalAdmin && (!isPremiumTier || !isAnyTeamSelected);

  const DEFAULT_SORT_HEADER = isPremiumTier ? "vulnerabilities" : "hosts_count";

  const initialQuery = (() => {
    let query = "";

    if (queryParams && queryParams.query) {
      query = queryParams.query;
    }

    return query;
  })();

  const initialSortHeader = (() => {
    let sortHeader = isPremiumTier ? "vulnerabilities" : "hosts_count";

    if (queryParams && queryParams.order_key) {
      sortHeader = queryParams.order_key;
    }

    return sortHeader;
  })();

  const initialSortDirection = ((): "asc" | "desc" | undefined => {
    let sortDirection = "desc";

    if (queryParams && queryParams.order_direction) {
      sortDirection = queryParams.order_direction;
    }

    return sortDirection as "asc" | "desc" | undefined;
  })();

  const initialPage = (() => {
    let page = 0;

    if (queryParams && queryParams.page) {
      page = parseInt(queryParams.page, 10);
    }

    return page;
  })();

  const initialVulnFilter = (() => {
    let isFilteredByVulnerabilities = false;

    if (queryParams && queryParams.vulnerable === "true") {
      isFilteredByVulnerabilities = true;
    }

    return isFilteredByVulnerabilities;
  })();

  const [filterVuln, setFilterVuln] = useState(initialVulnFilter);
  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(initialSortDirection);
  const [sortHeader, setSortHeader] = useState(initialSortHeader);
  const [page, setPage] = useState(initialPage);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryData>();
  const [resetPageIndex, setResetPageIndex] = useState<boolean>(false);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showPreviewTicketModal, setShowPreviewTicketModal] = useState(false);

  useEffect(() => {
    setFilterVuln(initialVulnFilter);
    setPage(initialPage);
    setSearchQuery(initialQuery);
    // TODO: handle invalid values for params
  }, [location]);

  useEffect(() => {
    const path = location.pathname + location.search;
    if (filteredSoftwarePath !== path) {
      setFilteredSoftwarePath(path);
    }
  }, [filteredSoftwarePath, location, setFilteredSoftwarePath]);

  // softwareConfig is either the global config or the team config of the currently selected team
  const {
    data: softwareConfig,
    error: softwareConfigError,
    isFetching: isFetchingSoftwareConfig,
    refetch: refetchSoftwareConfig,
  } = useQuery<
    IConfig | ILoadTeamResponse,
    Error,
    IConfig | ITeamConfig,
    ISoftwareConfigQueryKey[]
  >(
    [{ scope: "softwareConfig", teamId: teamIdForApi }],
    ({ queryKey }) => {
      const { teamId } = queryKey[0];
      return teamId ? teamsAPI.load(teamId) : configAPI.loadAll();
    },
    {
      enabled: isRouteOk,
      select: (data) => ("team" in data ? data.team : data),
    }
  );

  const isSoftwareConfigLoaded =
    !isFetchingSoftwareConfig && !softwareConfigError && !!softwareConfig;

  const isSoftwareEnabled = !!softwareConfig?.features
    ?.enable_software_inventory;

  const vulnWebhookSettings =
    softwareConfig?.webhook_settings?.vulnerabilities_webhook;

  const isVulnWebhookEnabled = !!vulnWebhookSettings?.enable_vulnerabilities_webhook;

  const isVulnIntegrationEnabled = (integrations?: IIntegrations) => {
    return (
      !!integrations?.jira?.some((j) => j.enable_software_vulnerabilities) ||
      !!integrations?.zendesk?.some((z) => z.enable_software_vulnerabilities)
    );
  };

  const isAnyVulnAutomationEnabled =
    isVulnWebhookEnabled ||
    isVulnIntegrationEnabled(softwareConfig?.integrations);

  const recentVulnerabilityMaxAge = (() => {
    let maxAgeInNanoseconds: number | undefined;
    if (softwareConfig && "vulnerabilities" in softwareConfig) {
      maxAgeInNanoseconds =
        softwareConfig.vulnerabilities.recent_vulnerability_max_age;
    } else {
      maxAgeInNanoseconds =
        globalConfig?.vulnerabilities.recent_vulnerability_max_age;
    }
    return maxAgeInNanoseconds
      ? Math.round(maxAgeInNanoseconds / 86400000000000) // convert from nanoseconds to days
      : CONFIG_DEFAULT_RECENT_VULNERABILITY_MAX_AGE_IN_DAYS;
  })();

  const {
    data: software,
    error: softwareError,
    isFetching: isFetchingSoftware,
  } = useQuery<
    ISoftwareResponse,
    Error,
    ISoftwareResponse,
    ISoftwareQueryKey[]
  >(
    [
      {
        scope: "software",
        page: tableQueryData?.pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDirection: sortDirection,
        // API expects "epss_probability" rather than "vulnerabilities"
        orderKey:
          isPremiumTier && sortHeader === "vulnerabilities"
            ? "epss_probability"
            : sortHeader,
        teamId: teamIdForApi,
        vulnerable: filterVuln,
      },
    ],
    ({ queryKey }) => softwareAPI.load(queryKey[0]),
    {
      enabled: isRouteOk && isSoftwareConfigLoaded,
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
    }
  );

  const {
    data: softwareCount,
    error: softwareCountError,
    isFetching: isFetchingCount,
  } = useQuery<ISoftwareCountResponse, Error, number, ISoftwareCountQueryKey[]>(
    [
      {
        scope: "softwareCount",
        query: searchQuery,
        vulnerable: filterVuln,
        teamId: teamIdForApi,
      },
    ],
    ({ queryKey }) => softwareAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk && isSoftwareConfigLoaded,
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  // NOTE: this is called once on initial render and every time the query changes
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      if (!isRouteOk || isEqual(newTableQuery, tableQueryData)) {
        return;
      }

      setTableQueryData({ ...newTableQuery });

      const {
        pageIndex,
        searchQuery: newSearchQuery,
        sortDirection: newSortDirection,
      } = newTableQuery;
      let { sortHeader: newSortHeader } = newTableQuery;

      pageIndex !== page && setPage(pageIndex);
      searchQuery !== newSearchQuery && setSearchQuery(newSearchQuery);
      sortDirection !== newSortDirection &&
        setSortDirection(
          newSortDirection === "asc" || newSortDirection === "desc"
            ? newSortDirection
            : DEFAULT_SORT_DIRECTION
        );

      if (isPremiumTier && newSortHeader === "vulnerabilities") {
        newSortHeader = "epss_probability";
      }
      sortHeader !== newSortHeader && setSortHeader(newSortHeader);

      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number | undefined } = {};
      if (!isEmpty(newSearchQuery)) {
        newQueryParams.query = newSearchQuery;
      }
      newQueryParams.page = pageIndex;
      newQueryParams.order_key = newSortHeader || DEFAULT_SORT_HEADER;
      newQueryParams.order_direction =
        newSortDirection || DEFAULT_SORT_DIRECTION;

      newQueryParams.vulnerable = filterVuln ? "true" : undefined;

      if (teamIdForApi !== undefined) {
        newQueryParams.team_id = teamIdForApi;
      }

      const locationPath = getNextLocationPath({
        pathPrefix: PATHS.MANAGE_SOFTWARE,
        routeTemplate,
        queryParams: newQueryParams,
      });
      router.replace(locationPath);
    },
    [
      isRouteOk,
      teamIdForApi,
      tableQueryData,
      page,
      searchQuery,
      sortDirection,
      isPremiumTier,
      sortHeader,
      DEFAULT_SORT_HEADER,
      filterVuln,
      routeTemplate,
      router,
    ]
  );

  const toggleManageAutomationsModal = useCallback(() => {
    setShowManageAutomationsModal(!showManageAutomationsModal);
  }, [setShowManageAutomationsModal, showManageAutomationsModal]);

  const togglePreviewPayloadModal = useCallback(() => {
    setShowPreviewPayloadModal(!showPreviewPayloadModal);
  }, [setShowPreviewPayloadModal, showPreviewPayloadModal]);

  const togglePreviewTicketModal = useCallback(() => {
    setShowPreviewTicketModal(!showPreviewTicketModal);
  }, [setShowPreviewTicketModal, showPreviewTicketModal]);

  const onCreateWebhookSubmit = async (
    configSoftwareAutomations: ISoftwareAutomations
  ) => {
    try {
      const request = configAPI.update(configSoftwareAutomations);
      await request.then(() => {
        renderFlash(
          "success",
          "Successfully updated vulnerability automations."
        );
        refetchSoftwareConfig();
      });
    } catch {
      renderFlash(
        "error",
        "Could not update vulnerability automations. Please try again."
      );
    } finally {
      toggleManageAutomationsModal();
    }
  };

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
      setPage(0);
    },
    [handleTeamChange]
  );

  // NOTE: used to reset page number to 0 when modifying filters
  const handleResetPageIndex = () => {
    setTableQueryData(
      (prevState) =>
        ({
          ...prevState,
          pageIndex: 0,
        } as ITableQueryData)
    );
    setResetPageIndex(true);
  };

  // NOTE: used to reset page number to 0 when modifying filters
  useEffect(() => {
    // TODO: cleanup this effect
    setResetPageIndex(false);
  }, [queryParams]);

  const renderHeaderDescription = () => {
    return (
      <p>
        Search for installed software{" "}
        {(isGlobalAdmin || isGlobalMaintainer) &&
          (!isPremiumTier || !isAnyTeamSelected) &&
          "and manage automations for detected vulnerabilities (CVEs)"}{" "}
        on{" "}
        <b>
          {isPremiumTier && isAnyTeamSelected
            ? "all hosts assigned to this team"
            : "all of your hosts"}
        </b>
        .
      </p>
    );
  };

  const renderSoftwareCount = useCallback(() => {
    const count = softwareCount;
    const lastUpdatedAt = software?.counts_updated_at;

    if (!isSoftwareEnabled || !lastUpdatedAt) {
      return null;
    }

    if (softwareCountError && !isFetchingCount) {
      return (
        <span className={`${baseClass}__count count-error`}>
          Failed to load software count
        </span>
      );
    }

    if (count) {
      return (
        <div
          className={`${baseClass}__count ${
            isFetchingCount ? "count-loading" : ""
          }`}
        >
          <span>{`${count} software item${count === 1 ? "" : "s"}`}</span>
          <LastUpdatedText
            lastUpdatedAt={lastUpdatedAt}
            whatToRetrieve={"software"}
          />
        </div>
      );
    }

    return null;
  }, [
    isFetchingCount,
    software,
    softwareCountError,
    softwareCount,
    isSoftwareEnabled,
  ]);

  const handleVulnFilterDropdownChange = (isFilterVulnerable: string) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_SOFTWARE,
        routeTemplate,
        queryParams: {
          ...queryParams,
          vulnerable: isFilterVulnerable,
          page: 0, // resets page index
        },
      })
    );
  };

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={filterVuln}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={handleVulnFilterDropdownChange}
        tableFilterDropdown
      />
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

  // TODO: Rework after backend is adjusted to differentiate empty search/filter results from
  // collecting inventory
  const isCollectingInventory =
    !searchQuery &&
    !filterVuln &&
    page === 0 &&
    !software?.software &&
    software?.counts_updated_at === null;

  const isLastPage =
    tableQueryData &&
    !!softwareCount &&
    DEFAULT_PAGE_SIZE * page + (software?.software?.length || 0) >=
      softwareCount;

  const softwareTableHeaders = useMemo(
    () =>
      generateSoftwareTableHeaders(
        router,
        isPremiumTier,
        isSandboxMode,
        currentTeamId
      ),
    [isPremiumTier, isSandboxMode, router, currentTeamId]
  );
  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = {
      software_id: row.original.id,
      team_id: currentTeamId,
    };

    const path = hostsBySoftwareParams
      ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
          hostsBySoftwareParams
        )}`
      : PATHS.MANAGE_HOSTS;

    router.push(path);
  };

  const searchable =
    isSoftwareEnabled &&
    (!!software?.software ||
      searchQuery !== "" ||
      queryParams.vulnerable === "true");

  const renderSoftwareTable = () => {
    if (
      isFetchingCount ||
      isFetchingSoftware ||
      !globalConfig ||
      (!softwareConfig && !softwareConfigError)
    ) {
      return <Spinner />;
    }
    if (
      (softwareError && !isFetchingSoftware) ||
      (softwareConfigError && !isFetchingSoftwareConfig)
    ) {
      return <TableDataError />;
    }
    return (
      <TableContainer
        columns={softwareTableHeaders}
        data={(isSoftwareEnabled && software?.software) || []}
        isLoading={false}
        resultsTitle={"software items"}
        emptyComponent={() => (
          <EmptySoftwareTable
            isSoftwareDisabled={!isSoftwareEnabled}
            isFilterVulnerable={filterVuln}
            isSandboxMode={isSandboxMode}
            isCollectingSoftware={isCollectingInventory}
            isSearching={searchQuery !== ""}
            noSandboxHosts={noSandboxHosts}
          />
        )}
        defaultSortHeader={sortHeader || DEFAULT_SORT_HEADER}
        defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
        defaultPageIndex={page || 0}
        defaultSearchQuery={searchQuery}
        manualSortBy
        pageSize={DEFAULT_PAGE_SIZE}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        resetPageIndex={resetPageIndex}
        disableNextPage={isLastPage}
        searchable={searchable}
        inputPlaceHolder="Search by name or vulnerabilities (CVEs)"
        onQueryChange={onQueryChange}
        additionalQueries={filterVuln ? "vulnerable" : ""} // additionalQueries serves as a trigger
        // for the useDeepEffect hook to fire onQueryChange for events happeing outside of
        // the TableContainer
        customControl={searchable ? renderVulnFilterDropdown : undefined}
        stackControls
        renderCount={renderSoftwareCount}
        renderFooter={renderTableFooter}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
      />
    );
  };

  return (
    <MainContent>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Software</h1>}
                {isPremiumTier &&
                  ((userTeams && userTeams.length > 1) || isOnGlobalTeam) && (
                    <TeamsDropdown
                      currentUserTeams={userTeams || []}
                      selectedTeamId={currentTeamId}
                      onChange={onTeamChange}
                      isSandboxMode={isSandboxMode}
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  userTeams &&
                  userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
              </div>
            </div>
          </div>
          {canManageAutomations && !softwareError && isSoftwareConfigLoaded && (
            <Button
              onClick={toggleManageAutomationsModal}
              className={`${baseClass}__manage-automations button`}
              variant="brand"
            >
              <span>Manage automations</span>
            </Button>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          {renderHeaderDescription()}
        </div>
        <div className={`${baseClass}__table`}>{renderSoftwareTable()}</div>
        {showManageAutomationsModal && (
          <ManageAutomationsModal
            onCancel={toggleManageAutomationsModal}
            onCreateWebhookSubmit={onCreateWebhookSubmit}
            togglePreviewPayloadModal={togglePreviewPayloadModal}
            togglePreviewTicketModal={togglePreviewTicketModal}
            showPreviewPayloadModal={showPreviewPayloadModal}
            showPreviewTicketModal={showPreviewTicketModal}
            softwareVulnerabilityAutomationEnabled={isAnyVulnAutomationEnabled}
            softwareVulnerabilityWebhookEnabled={isVulnWebhookEnabled}
            currentDestinationUrl={vulnWebhookSettings?.destination_url || ""}
            recentVulnerabilityMaxAge={recentVulnerabilityMaxAge}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManageSoftwarePage;
