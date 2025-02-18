import React, { useState, useEffect, useContext } from "react";
import { useQuery, useMutation } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IHost, IHostResponse } from "interfaces/host";
import { ILabel } from "interfaces/label";
import {
  IPolicyFormData,
  IPolicy,
  IStoredPolicyResponse,
} from "interfaces/policy";
import { ITarget } from "interfaces/target";
import {
  API_ALL_TEAMS_ID,
  APP_CONTEXT_ALL_TEAMS_ID,
  ITeam,
} from "interfaces/team";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import hostAPI from "services/entities/hosts";
import statusAPI from "services/entities/status";
import { DOCUMENT_TITLE_SUFFIX, LIVE_POLICY_STEPS } from "utilities/constants";

import QuerySidePanel from "components/side_panels/QuerySidePanel";
import QueryEditor from "pages/policies/PolicyPage/screens/QueryEditor";
import SelectTargets from "components/LiveQuery/SelectTargets";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import Spinner from "components/Spinner/Spinner";
import CustomLink from "components/CustomLink";
import RunQuery from "pages/policies/PolicyPage/screens/RunQuery";
import { DEFAULT_POLICY } from "pages/policies/constants";

interface IPolicyPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    search: string;
    query: { host_ids: string; team_id: string };
    hash?: string;
  };
}

const baseClass = "policy-page";

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

  const [step, setStep] = useState(LIVE_POLICY_STEPS[1]);
  const [selectedTargets, setSelectedTargets] = useState<ITarget[]>([]);
  const [targetedHosts, setTargetedHosts] = useState<IHost[]>([]);
  const [targetedLabels, setTargetedLabels] = useState<ILabel[]>([]);
  const [targetedTeams, setTargetedTeams] = useState<ITeam[]>([]);
  const [targetsTotalCount, setTargetsTotalCount] = useState(0);
  const [isLiveQueryRunnable, setIsLiveQueryRunnable] = useState(true);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [showOpenSchemaActionText, setShowOpenSchemaActionText] = useState(
    false
  );

  const {
    isLoading: isStoredPolicyLoading,
    data: storedPolicy,
    error: storedPolicyError,
  } = useQuery<IStoredPolicyResponse, Error, IPolicy>(
    ["policy", policyId],
    () => globalPoliciesAPI.load(policyId as number), // Note: Team users have access to policies through global API
    {
      enabled: isRouteOk && !!policyId, // Note: this justifies the number type assertions above
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

  useQuery<IHostResponse, Error, IHost>(
    "hostFromURL",
    () =>
      hostAPI.loadHostDetails(parseInt(location.query.host_ids as string, 10)), // TODO(sarah): What should happen if this doesn't parse (e.g. the string is "foo")? Also, note that "1,2,3" parses as 1.
    {
      enabled: isRouteOk && !!location.query.host_ids,
      retry: false,
      select: (data: IHostResponse) => data.host,
      onSuccess: (host) => {
        const targets = selectedTargets;
        host.target_type = "hosts";
        targets.push(host);
        setSelectedTargets([...targets]);
      },
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
    !(storedPolicy?.team_id?.toString() === location.query.team_id)
  ) {
    router.push(
      `${location.pathname}?team_id=${storedPolicy?.team_id?.toString()}`
    );
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
            Fleet is unable to run a live query. Refresh the page or log in
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
    const step1Opts = {
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
      goToSelectTargets: () => setStep(LIVE_POLICY_STEPS[2]),
      onOpenSchemaSidebar,
      renderLiveQueryWarning,
    };

    const step2Opts = {
      baseClass,
      selectedTargets,
      targetedHosts,
      targetedLabels,
      targetedTeams,
      targetsTotalCount,
      goToQueryEditor: () => setStep(LIVE_POLICY_STEPS[1]),
      goToRunQuery: () => setStep(LIVE_POLICY_STEPS[3]),
      setSelectedTargets,
      setTargetedHosts,
      setTargetedLabels,
      setTargetedTeams,
      setTargetsTotalCount,
      isLivePolicy: true,
    };

    const step3Opts = {
      selectedTargets,
      storedPolicy,
      setSelectedTargets,
      goToQueryEditor: () => setStep(LIVE_POLICY_STEPS[1]),
      targetsTotalCount,
    };

    switch (step) {
      case LIVE_POLICY_STEPS[2]:
        return <SelectTargets {...step2Opts} />;
      case LIVE_POLICY_STEPS[3]:
        return <RunQuery {...step3Opts} />;
      default:
        return <QueryEditor {...step1Opts} />;
    }
  };

  const isFirstStep = step === LIVE_POLICY_STEPS[1];
  const showSidebar =
    isFirstStep &&
    isSidebarOpen &&
    (isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin);

  if (!isRouteOk) {
    return <Spinner />;
  }

  return (
    <>
      <MainContent className={baseClass}>
        <div className={`${baseClass}__wrapper`}>{renderScreen()}</div>
      </MainContent>
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
  );
};

export default PolicyPage;
