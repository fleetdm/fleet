import React, { useCallback, useContext, useState } from "react";
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
  IIntegrations,
} from "interfaces/integration";
import { ITeamConfig } from "interfaces/team";
import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { buildQueryStringFromParams } from "utilities/url";

import Button from "components/buttons/Button";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import TabsWrapper from "components/TabsWrapper";

import ManageAutomationsModal from "./components/ManageSoftwareAutomationsModal";

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
      vulnerable?: string;
      exploited?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
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
    isPremiumTier,
    isSandboxMode,
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
  // TODO: move these down into the Software Titles component.
  const query = queryParams && queryParams.query ? queryParams.query : "";
  const showVulnerableSoftware =
    queryParams !== undefined && queryParams.vulnerable === "true";
  const showExploitedVulnerabilitiesOnly =
    queryParams !== undefined && queryParams.exploited === "true";

  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showPreviewTicketModal, setShowPreviewTicketModal] = useState(false);

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
  const isVulnIntegrationEnabled = (integrations?: IIntegrations) => {
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

  const canManageAutomations =
    isGlobalAdmin && (!isPremiumTier || !isAnyTeamSelected);

  const toggleManageAutomationsModal = useCallback(() => {
    setShowManageAutomationsModal(!showManageAutomationsModal);
  }, [setShowManageAutomationsModal, showManageAutomationsModal]);

  const togglePreviewPayloadModal = useCallback(() => {
    setShowPreviewPayloadModal(!showPreviewPayloadModal);
  }, [setShowPreviewPayloadModal, showPreviewPayloadModal]);

  const togglePreviewTicketModal = useCallback(() => {
    setShowPreviewTicketModal(!showPreviewTicketModal);
  }, [setShowPreviewTicketModal, showPreviewTicketModal]);

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

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
      // TODO: reset page to 0 when changing teams
    },
    [handleTeamChange]
  );

  const navigateToNav = useCallback(
    (i: number): void => {
      // Only query param to persist between tabs is team id
      const teamIdParam = buildQueryStringFromParams({
        team_id: location?.query.team_id,
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
            isSandboxMode={isSandboxMode}
          />
        )}
      </>
    );
  };

  const renderHeaderDescription = () => {
    return (
      <p>
        Search for installed software{" "}
        {(isGlobalAdmin || isGlobalMaintainer) &&
          (!isPremiumTier || !isAnyTeamSelected) &&
          "and manage automations for detected vulnerabilities (CVEs)"}{" "}
        on{" "}
        {isPremiumTier && isAnyTeamSelected
          ? "all hosts assigned to this team"
          : "all of your hosts"}
        .
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
          query,
          showVulnerableSoftware,
          showExploitedVulnerabilitiesOnly,
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
          {canManageAutomations && isSoftwareConfigLoaded && (
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
      </div>
    </MainContent>
  );
};

export default SoftwarePage;
