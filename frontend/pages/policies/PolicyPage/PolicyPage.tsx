import React, { useState, useEffect, useContext } from "react";
import { useQuery, useMutation } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { QUERIES_PAGE_STEPS } from "utilities/constants";
import { DEFAULT_POLICY } from "pages/policies/constants";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import hostAPI from "services/entities/hosts";
import statusAPI from "services/entities/status";
import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IPolicyFormData, IPolicy } from "interfaces/policy";
import { ITarget } from "interfaces/target";
import { ITeam } from "interfaces/team";

import QuerySidePanel from "components/side_panels/QuerySidePanel";
import QueryEditor from "pages/policies/PolicyPage/screens/QueryEditor";
import SelectTargets from "components/LiveQuery/SelectTargets";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import RunQuery from "pages/policies/PolicyPage/screens/RunQuery";
import ExternalLinkIcon from "../../../../assets/images/icon-external-link-12x12@2x.png";

interface IPolicyPageProps {
  router: InjectedRouter;
  params: Params;
  location: { query: { host_ids: string; team_id: string } };
}

interface IStoredPolicyResponse {
  policy: IPolicy;
}

interface IHostResponse {
  host: IHost;
}

const baseClass = "policy-page";

const PolicyPage = ({
  router,
  params: { id: paramsPolicyId },
  location: { query: URLQuerySearch },
}: IPolicyPageProps): JSX.Element => {
  const policyId = paramsPolicyId ? parseInt(paramsPolicyId, 10) : null;
  const policyTeamId = parseInt(URLQuerySearch.team_id, 10) || 0;
  const handlePageError = useErrorHandler();
  const {
    currentUser,
    currentTeam,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    setCurrentTeam,
  } = useContext(AppContext);
  const {
    lastEditedQueryBody,
    selectedOsqueryTable,
    setSelectedOsqueryTable,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryPlatform,
    setPolicyTeamId,
  } = useContext(PolicyContext);

  useEffect(() => {
    if (lastEditedQueryBody === "") {
      setLastEditedQueryBody(DEFAULT_POLICY.query);
    }
  }, []);

  useEffect(() => {
    // cleanup when component unmounts
    return () => {
      setLastEditedQueryPlatform(null);
    };
  }, []);

  if (currentUser && currentUser.teams.length && policyTeamId && !currentTeam) {
    const thisPolicyTeam = currentUser.teams.find(
      (team) => team.id === policyTeamId
    );
    if (thisPolicyTeam) {
      setCurrentTeam(thisPolicyTeam);
    }
  }

  const [step, setStep] = useState(QUERIES_PAGE_STEPS[1]);
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
    () =>
      policyTeamId
        ? teamPoliciesAPI.load(policyTeamId, policyId as number)
        : globalPoliciesAPI.load(policyId as number),
    {
      enabled: !!policyId,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IStoredPolicyResponse) => data.policy,
      onSuccess: (returnedQuery) => {
        setLastEditedQueryId(returnedQuery.id);
        setLastEditedQueryName(returnedQuery.name);
        setLastEditedQueryDescription(returnedQuery.description);
        setLastEditedQueryBody(returnedQuery.query);
        setLastEditedQueryResolution(returnedQuery.resolution);
        setLastEditedQueryPlatform(returnedQuery.platform);
        setPolicyTeamId(returnedQuery.team_id || 0);
      },
      onError: (error) => handlePageError(error),
    }
  );

  useQuery<IHostResponse, Error, IHost>(
    "hostFromURL",
    () =>
      hostAPI.loadHostDetails(parseInt(URLQuerySearch.host_ids as string, 10)),
    {
      enabled: !!URLQuerySearch.host_ids,
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

  const { mutateAsync: createPolicy } = useMutation(
    (formData: IPolicyFormData) => {
      return formData.team_id
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
    if (isLiveQueryRunnable) {
      return null;
    }

    return (
      <div className={`${baseClass}__warning`}>
        <div className={`${baseClass}__message`}>
          <p>
            Fleet is unable to run a live query. Refresh the page or log in
            again. If this keeps happening please{" "}
            <a
              target="_blank"
              rel="noopener noreferrer"
              href="https://github.com/fleetdm/fleet/issues/new/choose"
            >
              file an issue
              <img src={ExternalLinkIcon} alt="Open external link" />
            </a>
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
      storedPolicyError,
      createPolicy,
      onOsqueryTableSelect,
      goToSelectTargets: () => setStep(QUERIES_PAGE_STEPS[2]),
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
      goToQueryEditor: () => setStep(QUERIES_PAGE_STEPS[1]),
      goToRunQuery: () => setStep(QUERIES_PAGE_STEPS[3]),
      setSelectedTargets,
      setTargetedHosts,
      setTargetedLabels,
      setTargetedTeams,
      setTargetsTotalCount,
    };

    const step3Opts = {
      selectedTargets,
      storedPolicy,
      setSelectedTargets,
      goToQueryEditor: () => setStep(QUERIES_PAGE_STEPS[1]),
      targetsTotalCount,
    };

    switch (step) {
      case QUERIES_PAGE_STEPS[2]:
        return <SelectTargets {...step2Opts} />;
      case QUERIES_PAGE_STEPS[3]:
        return <RunQuery {...step3Opts} />;
      default:
        return <QueryEditor {...step1Opts} />;
    }
  };

  const isFirstStep = step === QUERIES_PAGE_STEPS[1];
  const showSidebar =
    isFirstStep &&
    isSidebarOpen &&
    (isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin);

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
