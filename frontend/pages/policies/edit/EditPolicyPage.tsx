import React, { useState, useEffect, useContext } from "react";
import { useQuery, useMutation } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import useTeamIdParam from "hooks/useTeamIdParam";
import {
  IPolicyFormData,
  IPolicy,
  IStoredPolicyResponse,
} from "interfaces/policy";
import { API_ALL_TEAMS_ID, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import statusAPI from "services/entities/status";
import PATHS from "router/paths";
import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import SidePanelPage from "components/SidePanelPage";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import QueryEditor from "pages/policies/edit/screens/QueryEditor";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import Spinner from "components/Spinner/Spinner";
import CustomLink from "components/CustomLink";
import { DEFAULT_POLICY } from "pages/policies/constants";

interface IPolicyPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    search: string;
    query: { fleet_id: string };
  };
}

const baseClass = "edit-policy-page";

const PolicyPage = ({
  router,
  params: { id: paramsPolicyId },
  location,
}: IPolicyPageProps): JSX.Element => {
  const policyId = paramsPolicyId ? parseInt(paramsPolicyId, 10) : null; // TODO(sarah): What should happen if this doesn't parse (e.g. the string is "foo")?
  const handlePageError = useErrorHandler();
  const {
    isOnGlobalTeam,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    config,
  } = useContext(AppContext);
  const {
    lastEditedQueryBody,
    policyTeamId,
    selectedOsqueryTable,
    setSelectedOsqueryTable,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
    setLastEditedQueryLabelsIncludeAny,
    setLastEditedQueryLabelsExcludeAny,
    setPolicyTeamId,
  } = useContext(PolicyContext);

  const {
    isRouteOk,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamObserver,
    teamIdForApi,
    isObserverPlus,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: true,
      observer_plus: true,
      technician: true,
    },
  });

  // // TODO(Sarah): What should happen if a user without save permissions tries to directly navigate
  // // to the new policy page? Should we redirect to the manage policies page?
  // const hasSavePermissions =
  //   isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
  //
  // useEffect(() => {
  //   if (!isRouteOk) {
  //     return;
  //   }
  //   if (trimEnd(location.pathname, "/").endsWith("/new")) {
  //     !hasSavePermissions && router.push(paths.MANAGE_POLICIES);
  //   }
  // }, [hasSavePermissions, isRouteOk, location.pathname, router]);

  useEffect(() => {
    if (!isRouteOk) {
      return;
    }
    if (policyTeamId !== teamIdForApi) {
      setPolicyTeamId(
        teamIdForApi === API_ALL_TEAMS_ID
          ? APP_CONTEXT_ALL_TEAMS_ID
          : teamIdForApi
      );
    }
  }, [isRouteOk, teamIdForApi, policyTeamId, setPolicyTeamId]);

  useEffect(() => {
    if (lastEditedQueryBody === "") {
      setLastEditedQueryBody(DEFAULT_POLICY.query);
    }
  }, []);

  useEffect(() => {
    // cleanup when component unmounts
    return () => {
      setLastEditedQueryCritical(false);
      setLastEditedQueryPlatform(null);
    };
  }, []);

  const [isLiveQueryRunnable, setIsLiveQueryRunnable] = useState(true);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [showOpenSchemaActionText, setShowOpenSchemaActionText] = useState(
    false
  );

  // TODO: Remove team endpoint workaround once global policy endpoint populates patch_software.
  // The global endpoint does not return patch_software for patch policies, but the team endpoint does.
  const {
    isLoading: isStoredPolicyLoading,
    data: storedPolicy,
    error: storedPolicyError,
  } = useQuery<IStoredPolicyResponse, Error, IPolicy>(
    ["policy", policyId, teamIdForApi],
    () =>
      teamIdForApi && teamIdForApi > 0
        ? teamPoliciesAPI.load(teamIdForApi, policyId as number)
        : globalPoliciesAPI.load(policyId as number),
    {
      enabled: isRouteOk && !!policyId,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IStoredPolicyResponse) => data.policy,
      onSuccess: (returnedQuery) => {
        const deNulledReturnedQueryTeamId = returnedQuery.team_id ?? undefined;

        setLastEditedQueryId(returnedQuery.id);
        setLastEditedQueryName(returnedQuery.name);
        setLastEditedQueryDescription(returnedQuery.description);
        setLastEditedQueryBody(returnedQuery.query);
        setLastEditedQueryResolution(returnedQuery.resolution);
        setLastEditedQueryCritical(returnedQuery.critical);
        setLastEditedQueryPlatform(returnedQuery.platform);
        setLastEditedQueryLabelsIncludeAny(
          returnedQuery.labels_include_any || []
        );
        setLastEditedQueryLabelsExcludeAny(
          returnedQuery.labels_exclude_any || []
        );
        // TODO(sarah): What happens if the team id in the policy response doesn't match the
        // url param? In theory, the backend should ensure this doesn't happen.
        setPolicyTeamId(
          deNulledReturnedQueryTeamId === API_ALL_TEAMS_ID
            ? APP_CONTEXT_ALL_TEAMS_ID
            : deNulledReturnedQueryTeamId
        );
      },
      onError: (error) => handlePageError(error),
    }
  );

  /** Pesky bug affecting team level users:
   - Navigating to policies/:id immediately defaults the user to the first team they're on
  with the most permissions, in the URL bar because of useTeamIdParam
  even if the policies/:id entity has a team attached to it
  Hacky fix:
   - Push entity's team id to url for team level users
  */
  if (
    !isOnGlobalTeam &&
    !isStoredPolicyLoading &&
    storedPolicy?.team_id !== undefined &&
    storedPolicy?.team_id !== null &&
    !(storedPolicy?.team_id?.toString() === location.query.fleet_id)
  ) {
    router.push(
      getPathWithQueryParams(location.pathname, {
        fleet_id: storedPolicy?.team_id?.toString(),
      })
    );
  }

  // Fetch team config to determine "Other" automations (webhooks/integrations)
  const { data: teamData } = useQuery<ILoadTeamResponse, Error>(
    ["teams", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled:
        isRouteOk &&
        teamIdForApi !== undefined &&
        teamIdForApi > 0 &&
        storedPolicy?.type === "patch",
      staleTime: 5000,
    }
  );

  let currentAutomatedPolicies: number[] = [];
  if (teamData?.team) {
    const {
      webhook_settings: { failing_policies_webhook: webhook },
      integrations,
    } = teamData.team;
    const isIntegrationEnabled =
      (integrations?.jira?.some((j: any) => j.enable_failing_policies) ||
        integrations?.zendesk?.some((z: any) => z.enable_failing_policies)) ??
      false;
    if (isIntegrationEnabled || webhook?.enable_failing_policies_webhook) {
      currentAutomatedPolicies = webhook?.policy_ids || [];
    }
  }

  // this function is passed way down, wrapped and ultimately called by SaveNewPolicyModal
  const { mutateAsync: createPolicy } = useMutation(
    (formData: IPolicyFormData) => {
      return formData.team_id !== undefined
        ? teamPoliciesAPI.create(formData)
        : globalPoliciesAPI.create(formData);
    }
  );

  const detectIsFleetQueryRunnable = () => {
    statusAPI.live_query().catch(() => {
      setIsLiveQueryRunnable(false);
    });
  };

  useEffect(() => {
    detectIsFleetQueryRunnable();
  }, []);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Antivirus healthy (Linux) | Policies | Fleet
    if (storedPolicy?.name) {
      document.title = `${storedPolicy.name} | Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, storedPolicy?.name]);

  useEffect(() => {
    setShowOpenSchemaActionText(!isSidebarOpen);
  }, [isSidebarOpen]);

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const onCloseSchemaSidebar = () => {
    setIsSidebarOpen(false);
  };

  const onOpenSchemaSidebar = () => {
    setIsSidebarOpen(true);
  };

  const renderLiveQueryWarning = (): JSX.Element | null => {
    if (isLiveQueryRunnable || config?.server_settings.live_query_disabled) {
      return null;
    }

    return (
      <div className={`${baseClass}__warning`}>
        <div className={`${baseClass}__message`}>
          <p>
            Fleet is unable to run a live report. Refresh the page or log in
            again. If this keeps happening please{" "}
            <CustomLink
              url="https://github.com/fleetdm/fleet/issues/new/choose"
              text="file an issue"
              newTab
            />
          </p>
        </div>
      </div>
    );
  };

  const renderScreen = () => {
    const queryEditorOpts = {
      router,
      baseClass,
      policyIdForEdit: policyId,
      showOpenSchemaActionText,
      storedPolicy,
      isStoredPolicyLoading,
      isTeamAdmin,
      isTeamMaintainer,
      isTeamObserver,
      isObserverPlus,
      storedPolicyError,
      createPolicy,
      onOsqueryTableSelect,
      goToSelectTargets: () =>
        router.push(
          getPathWithQueryParams(PATHS.LIVE_POLICY(policyId), {
            fleet_id: teamIdForApi,
          })
        ),
      onOpenSchemaSidebar,
      renderLiveQueryWarning,
      teamIdForApi,
      currentAutomatedPolicies,
    };

    return <QueryEditor {...queryEditorOpts} />;
  };

  const showSidebar =
    isSidebarOpen &&
    (isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin);

  if (!isRouteOk) {
    return <Spinner />;
  }

  return (
    <SidePanelPage>
      <>
        <MainContent className={baseClass}>{renderScreen()}</MainContent>
        {showSidebar && (
          <SidePanelContent>
            <QuerySidePanel
              onOsqueryTableSelect={onOsqueryTableSelect}
              selectedOsqueryTable={selectedOsqueryTable}
              onClose={onCloseSchemaSidebar}
            />
          </SidePanelContent>
        )}
      </>
    </SidePanelPage>
  );
};

export default PolicyPage;
