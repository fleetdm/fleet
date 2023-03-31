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
import { useDebouncedCallback } from "use-debounce";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
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
import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook"; // @ts-ignore
import configAPI from "services/entities/config";
import softwareAPI from "services/entities/software";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import {
  GITHUB_NEW_ISSUE_LINK,
  VULNERABLE_DROPDOWN_OPTIONS,
} from "utilities/constants";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import Spinner from "components/Spinner";
import TableContainer, { ITableQueryData } from "components/TableContainer";
import TableDataError from "components/DataError";
import TeamsDropdownHeader, {
  ITeamsDropdownState,
} from "components/PageHeader/TeamsDropdownHeader";
import LastUpdatedText from "components/LastUpdatedText";
import MainContent from "components/MainContent";
import CustomLink from "components/CustomLink";
import EmptySoftwareTable from "../components/EmptySoftwareTable";

import generateSoftwareTableHeaders from "./SoftwareTableConfig";
import ManageAutomationsModal from "./components/ManageAutomationsModal";

interface IManageSoftwarePageProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: { vulnerable?: string };
    search: string;
  };
}

interface ISoftwareQueryKey {
  scope: string;
  page: number;
  perPage: number;
  query: string;
  orderKey: string;
  orderDir?: "asc" | "desc";
  vulnerable: boolean;
  teamId?: number;
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
interface IHeaderButtonsState extends ITeamsDropdownState {
  isLoading: boolean;
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
  router,
  location,
}: IManageSoftwarePageProps): JSX.Element => {
  const {
    availableTeams,
    config: globalConfig,
    currentTeam,
    isOnGlobalTeam,
    isPremiumTier,
    isSandboxMode,
    noSandboxHosts,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const DEFAULT_SORT_HEADER = isPremiumTier ? "vulnerabilities" : "hosts_count";

  // TODO: refactor usage of vulnerable query param in accordance with new patterns for query params
  // and management of URL state
  const [filterVuln, setFilterVuln] = useState(
    location?.query?.vulnerable === "true" || false
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(DEFAULT_SORT_DIRECTION);
  const [sortHeader, setSortHeader] = useState(DEFAULT_SORT_HEADER);
  const [pageIndex, setPageIndex] = useState(0);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showPreviewTicketModal, setShowPreviewTicketModal] = useState(false);

  useEffect(() => {
    setFilterVuln(location?.query?.vulnerable === "true" || false);
    // TODO: handle invalid values for vulnerable param
  }, [location]);

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
    [{ scope: "softwareConfig", teamId: currentTeam?.id }],
    ({ queryKey }) => {
      const { teamId } = queryKey[0];
      return teamId ? teamsAPI.load(teamId) : configAPI.loadAll();
    },
    {
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
        page: pageIndex,
        perPage: DEFAULT_PAGE_SIZE,
        query: searchQuery,
        orderDir: sortDirection || DEFAULT_SORT_DIRECTION,
        // API expects "epss_probability" rather than "vulnerabilities"
        orderKey:
          isPremiumTier && sortHeader === "vulnerabilities"
            ? "epss_probability"
            : sortHeader,
        teamId: currentTeam?.id,
        vulnerable: !!location.query.vulnerable,
      },
    ],
    ({ queryKey }) => softwareAPI.load(queryKey[0]),
    {
      enabled:
        isSoftwareConfigLoaded &&
        (isOnGlobalTeam ||
          !!availableTeams?.find((t) => t.id === currentTeam?.id)),
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
    }
  );

  const {
    data: softwareCount,
    error: softwareCountError,
    isFetching: isFetchingCount,
  } = useQuery<
    ISoftwareCountResponse,
    Error,
    number,
    Partial<ISoftwareQueryKey>[]
  >(
    [
      {
        scope: "softwareCount",
        query: searchQuery,
        vulnerable: !!location.query.vulnerable,
        teamId: currentTeam?.id,
      },
    ],
    ({ queryKey }) => {
      return softwareAPI.count(queryKey[0]);
    },
    {
      enabled:
        isSoftwareConfigLoaded &&
        (isOnGlobalTeam ||
          !!availableTeams?.find((t) => t.id === currentTeam?.id)),
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  const onQueryChange = useDebouncedCallback(
    async ({
      pageIndex: newPageIndex,
      searchQuery: newSearchQuery,
      sortDirection: newSortDirection,
      sortHeader: newSortHeader,
    }: ITableQueryData) => {
      pageIndex !== newPageIndex && setPageIndex(newPageIndex);
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
    },
    300
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

  const onTeamSelect = () => {
    setPageIndex(0);
  };

  // TODO: refactor/replace team dropdown header component in accordance with new patterns
  const renderHeaderButtons = useCallback(
    (state: IHeaderButtonsState): JSX.Element | null => {
      const {
        teamId,
        isLoading,
        isGlobalAdmin,
        isPremiumTier: isPremium,
      } = state;
      const canManageAutomations =
        isGlobalAdmin && (!isPremium || teamId === 0);
      if (canManageAutomations && !softwareError && !isLoading) {
        return (
          <Button
            onClick={toggleManageAutomationsModal}
            className={`${baseClass}__manage-automations button`}
            variant="brand"
          >
            <span>Manage automations</span>
          </Button>
        );
      }
      return null;
    },
    [softwareError, toggleManageAutomationsModal]
  );

  // TODO: refactor/replace team dropdown header component in accordance with new patterns
  const renderHeaderDescription = (state: ITeamsDropdownState) => {
    return (
      <p>
        Search for installed software{" "}
        {(state.isGlobalAdmin || state.isGlobalMaintainer) &&
          (!state.isPremiumTier || state.teamId === 0) &&
          "and manage automations for detected vulnerabilities (CVEs)"}{" "}
        on{" "}
        <b>
          {state.isPremiumTier && !!state.teamId
            ? "all hosts assigned to this team"
            : "all of your hosts"}
        </b>
        .
      </p>
    );
  };

  // TODO: refactor/replace team dropdown header component in accordance with new patterns
  const renderHeader = useCallback(() => {
    return (
      <TeamsDropdownHeader
        location={location}
        router={router}
        baseClass={baseClass}
        onChange={onTeamSelect}
        defaultTitle="Software"
        description={renderHeaderDescription}
        buttons={(state) =>
          renderHeaderButtons({
            ...state,
            isLoading: !isSoftwareConfigLoaded,
          })
        }
      />
    );
  }, [router, location, isSoftwareConfigLoaded, renderHeaderButtons]);

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

  // TODO: refactor in accordance with new patterns for query params and management of URL state
  const buildUrlQueryString = (queryString: string, vulnerable: boolean) => {
    queryString = queryString.startsWith("?")
      ? queryString.slice(1)
      : queryString;
    const queryParams = queryString.split("&").filter((el) => el.includes("="));
    const index = queryParams.findIndex((el) => el.includes("vulnerable"));

    if (vulnerable) {
      const vulnParam = `vulnerable=${vulnerable}`;
      if (index >= 0) {
        // replace old vuln param
        queryParams.splice(index, 1, vulnParam);
      } else {
        // add new vuln param
        queryParams.push(vulnParam);
      }
    } else {
      // remove old vuln param
      index >= 0 && queryParams.splice(index, 1);
    }
    queryString = queryParams.length ? "?".concat(queryParams.join("&")) : "";

    return queryString;
  };

  // TODO: refactor in accordance with new patterns for query params and management of URL state
  const onVulnFilterChange = useCallback(
    (vulnerable: boolean) => {
      setFilterVuln(vulnerable);
      setPageIndex(0);
      const queryString = buildUrlQueryString(location?.search, vulnerable);
      if (location?.search !== queryString) {
        const path = location?.pathname?.concat(queryString);
        !!path && router.replace(path);
      }
    },
    [location, router]
  );

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={filterVuln}
        className={`${baseClass}__vuln_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={onVulnFilterChange}
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
    !currentTeam?.id &&
    !pageIndex &&
    !software?.software &&
    software?.counts_updated_at === null;

  const isLastPage =
    !!softwareCount &&
    DEFAULT_PAGE_SIZE * pageIndex + (software?.software?.length || 0) >=
      softwareCount;

  const softwareTableHeaders = useMemo(
    () => generateSoftwareTableHeaders(router, isPremiumTier),
    [isPremiumTier, router]
  );
  const handleRowSelect = (row: IRowProps) => {
    const queryParams = { software_id: row.original.id };

    const path = queryParams
      ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(queryParams)}`
      : PATHS.MANAGE_HOSTS;

    router.push(path);
  };

  const searchable =
    isSoftwareEnabled && (!!software?.software || searchQuery !== "");

  return !availableTeams ||
    !globalConfig ||
    (!softwareConfig && !softwareConfigError) ? (
    <Spinner />
  ) : (
    <MainContent>
      <div className={`${baseClass}__wrapper`}>
        {renderHeader()}
        <div className={`${baseClass}__table`}>
          {(softwareError && !isFetchingSoftware) ||
          (softwareConfigError && !isFetchingSoftwareConfig) ? (
            <TableDataError />
          ) : (
            <TableContainer
              columns={softwareTableHeaders}
              data={(isSoftwareEnabled && software?.software) || []}
              isLoading={isFetchingSoftware || isFetchingCount}
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
              defaultSortHeader={DEFAULT_SORT_HEADER}
              defaultSortDirection={DEFAULT_SORT_DIRECTION}
              manualSortBy
              pageSize={DEFAULT_PAGE_SIZE}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              disableNextPage={isLastPage}
              searchable={searchable}
              inputPlaceHolder="Search software by name or vulnerabilities (CVEs)"
              onQueryChange={onQueryChange}
              additionalQueries={filterVuln ? "vulnerable" : ""} // additionalQueries serves as a trigger
              // for the useDeepEffect hook to fire onQueryChange for events happeing outside of
              // the TableContainer
              customControl={searchable ? renderVulnFilterDropdown : undefined}
              stackControls
              renderCount={renderSoftwareCount}
              renderFooter={renderTableFooter}
              disableActionButton
              hideActionButton
              disableMultiRowSelect
              onSelectSingleRow={handleRowSelect}
            />
          )}
        </div>
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
