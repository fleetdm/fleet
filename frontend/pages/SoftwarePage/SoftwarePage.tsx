import React, { useCallback, useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { Tab, TabList, Tabs } from "react-tabs";

import PATHS from "router/paths";
import {
  IConfig,
  CONFIG_DEFAULT_RECENT_VULNERABILITY_MAX_AGE_IN_DAYS,
} from "interfaces/config";
import {
  IJiraIntegration,
  IZendeskIntegration,
  IZendeskJiraIntegrations,
} from "interfaces/integration";
import { APP_CONTEXT_ALL_TEAMS_ID, ITeamConfig } from "interfaces/team";
import { SelectedPlatform } from "interfaces/platform";
import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { ISoftwareApiParams } from "services/entities/software";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
} from "utilities/url";
import { getNextLocationPath } from "utilities/helpers";

import Button from "components/buttons/Button";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import TabsWrapper from "components/TabsWrapper";

import ManageAutomationsModal from "./components/ManageSoftwareAutomationsModal";
import AddSoftwareModal from "./components/AddSoftwareModal";
import {
  buildSoftwareFilterQueryParams,
  buildSoftwareVulnFiltersQueryParams,
  getSoftwareFilterFromQueryParams,
  getSoftwareVulnFiltersFromQueryParams,
  ISoftwareVulnFiltersParams,
} from "./SoftwareTitles/SoftwareTable/helpers";
import SoftwareFiltersModal from "./components/SoftwareFiltersModal";

interface ISoftwareSubNavItem {
  name: string;
  pathname: string;
}

const softwareSubNav: ISoftwareSubNavItem[] = [
  {
    name: "Software",
    pathname: PATHS.SOFTWARE_TITLES,
  },
  {
    name: "OS",
    pathname: PATHS.SOFTWARE_OS,
  },
  {
    name: "Vulnerabilities",
    pathname: PATHS.SOFTWARE_VULNERABILITIES,
  },
];

const getTabIndex = (path: string): number => {
  return softwareSubNav.findIndex((navItem) => {
    // This check ensures that for software versions path we still
    // highlight the software tab.
    if (navItem.name === "Software" && PATHS.SOFTWARE_VERSIONS === path) {
      return true;
    }
    // tab stays highlighted for paths that start with same pathname
    return path.startsWith(navItem.pathname);
  });
};

// default values for query params used on this page if not provided
const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_PAGE_SIZE = 20;
const DEFAULT_PAGE = 0;

const baseClass = "software-page";

interface ISoftwareAutomations {
  webhook_settings: {
    vulnerabilities_webhook: IWebhookSoftwareVulnerabilities;
  };
  integrations: {
    jira: IJiraIntegration[];
    zendesk: IZendeskIntegration[];
  };
}

interface ISoftwareConfigQueryKey {
  scope: string;
  teamId?: number;
}

interface ISoftwarePageProps {
  children: JSX.Element;
  location: {
    pathname: string;
    search: string;
    query: {
      team_id?: string;
      available_for_install?: string;
      vulnerable?: string;
      exploit?: string;
      min_cvss_score?: string;
      max_cvss_score?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
      platform?: SelectedPlatform;
    };
    hash?: string;
  };
  router: InjectedRouter; // v3
}

const SoftwarePage = ({ children, router, location }: ISoftwarePageProps) => {
  const {
    config: globalConfig,
    isFreeTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isTeamAdmin,
    isTeamMaintainer,
    isPremiumTier,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const queryParams = location.query;

  // initial values for query params used on this page
  const sortHeader =
    queryParams && queryParams.order_key
      ? queryParams.order_key
      : DEFAULT_SORT_HEADER;
  const sortDirection =
    queryParams?.order_direction === undefined
      ? DEFAULT_SORT_DIRECTION
      : queryParams.order_direction;
  const page =
    queryParams && queryParams.page
      ? parseInt(queryParams.page, 10)
      : DEFAULT_PAGE;
  const platform = queryParams?.platform || "all";
  // TODO: move these down into the Software Titles component.
  const query = queryParams && queryParams.query ? queryParams.query : "";
  const showExploitedVulnerabilitiesOnly =
    queryParams !== undefined && queryParams.exploit === "true";

  // TODO: there should be better validation of the params depending on the route (e.g., self_service
  // and available_for_install don't apply to versions, os, or vulnerabilities routes) and some
  // defined redirect behavior if the params are invalid
  const softwareFilter = getSoftwareFilterFromQueryParams(queryParams);

  const softwareVulnFilters = getSoftwareVulnFiltersFromQueryParams(
    queryParams
  );

  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showPreviewTicketModal, setShowPreviewTicketModal] = useState(false);
  const [showAddSoftwareModal, setShowAddSoftwareModal] = useState(false);
  const [showSoftwareFiltersModal, setShowSoftwareFiltersModal] = useState(
    false
  );
  const [resetPageIndex, setResetPageIndex] = useState<boolean>(false);
  const [addedSoftwareToken, setAddedSoftwareToken] = useState<string | null>(
    null
  );

  const {
    currentTeamId,
    isAllTeamsSelected,
    isRouteOk,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
  });

  // softwareConfig is either the global config or the team config of the
  // currently selected team depending on the page team context selected
  // by the user.
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

  // TODO: move into manage automations modal
  const vulnWebhookSettings =
    softwareConfig?.webhook_settings?.vulnerabilities_webhook;
  const isVulnWebhookEnabled = !!vulnWebhookSettings?.enable_vulnerabilities_webhook;
  const isVulnIntegrationEnabled = (
    integrations?: IZendeskJiraIntegrations
  ) => {
    return (
      !!integrations?.jira?.some((j) => j.enable_software_vulnerabilities) ||
      !!integrations?.zendesk?.some((z) => z.enable_software_vulnerabilities)
    );
  };

  // TODO: move into manage automations modal
  const isAnyVulnAutomationEnabled =
    isVulnWebhookEnabled ||
    isVulnIntegrationEnabled(softwareConfig?.integrations);

  // TODO: move into manage automations modal
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

  const isSoftwareConfigLoaded =
    !isFetchingSoftwareConfig && !softwareConfigError && !!softwareConfig;

  const toggleManageAutomationsModal = useCallback(() => {
    setShowManageAutomationsModal(!showManageAutomationsModal);
  }, [setShowManageAutomationsModal, showManageAutomationsModal]);

  const togglePreviewPayloadModal = useCallback(() => {
    setShowPreviewPayloadModal(!showPreviewPayloadModal);
  }, [setShowPreviewPayloadModal, showPreviewPayloadModal]);

  const togglePreviewTicketModal = useCallback(() => {
    setShowPreviewTicketModal(!showPreviewTicketModal);
  }, [setShowPreviewTicketModal, showPreviewTicketModal]);

  const toggleSoftwareFiltersModal = useCallback(() => {
    setShowSoftwareFiltersModal(!showSoftwareFiltersModal);
  }, [setShowSoftwareFiltersModal, showSoftwareFiltersModal]);

  // TODO: move into manage automations modal
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

  const onAddSoftware = useCallback(() => {
    if (currentTeamId === APP_CONTEXT_ALL_TEAMS_ID) {
      setShowAddSoftwareModal(true);
    } else {
      router.push(
        `${PATHS.SOFTWARE_ADD_FLEET_MAINTAINED}?team_id=${currentTeamId}`
      );
    }
  }, [currentTeamId, router]);

  // Used to reset page number to 0 when modifying filters
  // Solution reused from ManageHostPage.tsx
  useEffect(() => {
    setResetPageIndex(false);
  }, [queryParams, page]);

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
      // Used to reset page number to 0 when modifying filters
      setResetPageIndex(true);
    },
    [handleTeamChange]
  );

  const onApplyVulnFilters = (vulnFilters: ISoftwareVulnFiltersParams) => {
    const newQueryParams: ISoftwareApiParams = {
      query,
      teamId: currentTeamId,
      orderDirection: sortDirection,
      orderKey: sortHeader,
      page: 0, // resets page index
      ...buildSoftwareFilterQueryParams(softwareFilter),
      ...buildSoftwareVulnFiltersQueryParams(vulnFilters),
    };

    router.replace(
      getNextLocationPath({
        pathPrefix: location.pathname,
        routeTemplate: "",
        queryParams: convertParamsToSnakeCase(newQueryParams),
      })
    );
    toggleSoftwareFiltersModal();
  };

  const navigateToNav = useCallback(
    (i: number): void => {
      setResetPageIndex(true); // Fixes flakey page reset in table state when switching between tabs

      // Only query param to persist between tabs is team id
      const teamIdParam = buildQueryStringFromParams({
        team_id: location?.query.team_id,
        page: 0, // Fixes flakey page reset in API call when switching between tabs
      });

      const navPath = softwareSubNav[i].pathname.concat(`?${teamIdParam}`);
      router.replace(navPath);
    },
    [location, router]
  );

  const renderTitle = () => {
    return (
      <>
        {isFreeTier && <h1>Software</h1>}
        {isPremiumTier && (
          <TeamsHeader
            isOnGlobalTeam={isOnGlobalTeam}
            currentTeamId={currentTeamId}
            userTeams={userTeams}
            onTeamChange={onTeamChange}
          />
        )}
      </>
    );
  };

  const renderPageActions = () => {
    const canManageAutomations =
      isGlobalAdmin && (!isPremiumTier || isAllTeamsSelected);

    const canAddSoftware =
      isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

    if (!isSoftwareConfigLoaded) return null;

    return (
      <div className={`${baseClass}__action-buttons`}>
        {canManageAutomations && (
          <Button
            onClick={toggleManageAutomationsModal}
            className={`${baseClass}__manage-automations`}
            variant="inverse"
          >
            Manage automations
          </Button>
        )}
        {canAddSoftware && (
          <Button onClick={onAddSoftware} variant="brand">
            <span>Add software</span>
          </Button>
        )}
      </div>
    );
  };

  const renderHeaderDescription = () => {
    return (
      <p>
        Manage software and search for installed software, OS, and
        vulnerabilities {isAllTeamsSelected ? "for all hosts" : "on this team"}.
      </p>
    );
  };

  const renderBody = () => {
    return (
      <div>
        <TabsWrapper>
          <Tabs
            selectedIndex={getTabIndex(location?.pathname || "")}
            onSelect={navigateToNav}
          >
            <TabList>
              {softwareSubNav.map((navItem) => {
                return (
                  <Tab key={navItem.name} data-text={navItem.name}>
                    {navItem.name}
                  </Tab>
                );
              })}
            </TabList>
          </Tabs>
        </TabsWrapper>
        {React.cloneElement(children, {
          router,
          isSoftwareEnabled: Boolean(
            softwareConfig?.features?.enable_software_inventory
          ),
          perPage: DEFAULT_PAGE_SIZE,
          orderDirection: sortDirection,
          orderKey: sortHeader,
          currentPage: page,
          teamId: teamIdForApi,
          // TODO: move down into the Software Titles component
          platform,
          query,
          showExploitedVulnerabilitiesOnly,
          softwareFilter,
          vulnFilters: softwareVulnFilters,
          resetPageIndex,
          addedSoftwareToken,
          onAddFiltersClick: toggleSoftwareFiltersModal,
        })}
      </div>
    );
  };

  return (
    <MainContent>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>{renderTitle()}</div>
            </div>
          </div>
          {renderPageActions()}
        </div>
        <div className={`${baseClass}__description`}>
          {renderHeaderDescription()}
        </div>
        {renderBody()}
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
        {showAddSoftwareModal && (
          <AddSoftwareModal
            onExit={() => setShowAddSoftwareModal(false)}
            isFreeTier={isFreeTier}
          />
        )}
        {showSoftwareFiltersModal && (
          <SoftwareFiltersModal
            onExit={toggleSoftwareFiltersModal}
            onSubmit={onApplyVulnFilters}
            vulnFilters={softwareVulnFilters}
            isPremiumTier={isPremiumTier || false}
          />
        )}
      </div>
    </MainContent>
  );
};

export default SoftwarePage;
