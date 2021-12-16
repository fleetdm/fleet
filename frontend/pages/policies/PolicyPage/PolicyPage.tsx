import React, { useState, useEffect, useContext } from "react";
import { useQuery, useMutation } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";

// @ts-ignore
import Fleet from "fleet"; // @ts-ignore
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { QUERIES_PAGE_STEPS, DEFAULT_POLICY } from "utilities/constants";
import globalPoliciesAPI from "services/entities/global_policies"; // @ts-ignore
import teamPoliciesAPI from "services/entities/team_policies"; // @ts-ignore
import hostAPI from "services/entities/hosts"; // @ts-ignore
import { IPolicyFormData, IPolicy } from "interfaces/policy";
import { ITarget } from "interfaces/target";
import { IHost } from "interfaces/host";

import QuerySidePanel from "components/side_panels/QuerySidePanel";
import QueryEditor from "pages/policies/PolicyPage/screens/QueryEditor";
import SelectTargets from "pages/policies/PolicyPage/screens/SelectTargets";
import RunQuery from "pages/policies/PolicyPage/screens/RunQuery";
import ExternalURLIcon from "../../../../assets/images/icon-external-url-12x12@2x.png";

interface IPolicyPageProps {
  router: InjectedRouter;
  params: Params;
  location: any; // no type in react-router v3
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
  const policyIdForEdit = paramsPolicyId ? parseInt(paramsPolicyId, 10) : null;
  const policyTeamId = URLQuerySearch.team_id || 0;
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

  if (
    policyTeamId &&
    currentUser &&
    !currentUser.teams.length &&
    !currentTeam
  ) {
    const thisPolicyTeam = currentUser.teams.find((team) => team.id);
    if (thisPolicyTeam) {
      setCurrentTeam(thisPolicyTeam);
    }
  }

  const [step, setStep] = useState<string>(QUERIES_PAGE_STEPS[1]);
  const [selectedTargets, setSelectedTargets] = useState<ITarget[]>([]);
  const [isLiveQueryRunnable, setIsLiveQueryRunnable] = useState<boolean>(true);
  const [isSidebarOpen, setIsSidebarOpen] = useState<boolean>(true);
  const [
    showOpenSchemaActionText,
    setShowOpenSchemaActionText,
  ] = useState<boolean>(false);

  // disabled on page load so we can control the number of renders
  // else it will re-populate the context on occasion
  const {
    isLoading: isStoredPolicyLoading,
    data: storedPolicy,
    error: storedPolicyError,
    refetch: refetchStoredPolicy,
  } = useQuery<IStoredPolicyResponse, Error, IPolicy>(
    ["query", policyIdForEdit],
    () =>
      policyTeamId
        ? teamPoliciesAPI.load(policyTeamId, policyIdForEdit as number)
        : globalPoliciesAPI.load(policyIdForEdit as number),
    {
      enabled: false,
      refetchOnWindowFocus: false,
      select: (data: IStoredPolicyResponse) => data.policy,
      onSuccess: (returnedQuery) => {
        setLastEditedQueryName(returnedQuery.name);
        setLastEditedQueryDescription(returnedQuery.description);
        setLastEditedQueryBody(returnedQuery.query);
        setLastEditedQueryResolution(returnedQuery.resolution);
        setLastEditedQueryPlatform(returnedQuery.platform);
        setPolicyTeamId(returnedQuery.team_id || 0);
      },
    }
  );

  // if URL is like `/policies/1?host_ids=22`, add the host
  // to the selected targets automatically
  useQuery<IHostResponse, Error, IHost>(
    "hostFromURL",
    () => hostAPI.load(parseInt(URLQuerySearch.host_ids as string, 10)),
    {
      enabled: !!URLQuerySearch.host_ids,
      select: (data: IHostResponse) => data.host,
      onSuccess: (data) => {
        const targets = selectedTargets;
        const hostTarget = data as any; // intentional so we can add to the object

        hostTarget.target_type = "hosts";

        targets.push(hostTarget as IHost);
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

  useEffect(() => {
    const detectIsFleetQueryRunnable = () => {
      Fleet.status.live_query().catch(() => {
        setIsLiveQueryRunnable(false);
      });
    };

    detectIsFleetQueryRunnable();
    !!policyIdForEdit && refetchStoredPolicy();
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
              file an issue <img alt="" src={ExternalURLIcon} />
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
      policyIdForEdit,
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
      selectedTargets: [...selectedTargets],
      goToQueryEditor: () => setStep(QUERIES_PAGE_STEPS[1]),
      goToRunQuery: () => setStep(QUERIES_PAGE_STEPS[3]),
      setSelectedTargets,
    };

    const step3Opts = {
      selectedTargets,
      storedPolicy,
      policyIdForEdit,
      setSelectedTargets,
      goToQueryEditor: () => setStep(QUERIES_PAGE_STEPS[1]),
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
  const sidebarClass = isFirstStep && isSidebarOpen && "has-sidebar";
  const showSidebar =
    isFirstStep &&
    isSidebarOpen &&
    (isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin);

  return (
    <div className={`${baseClass} ${sidebarClass}`}>
      <div className={`${baseClass}__content`}>{renderScreen()}</div>
      {showSidebar && (
        <QuerySidePanel
          onOsqueryTableSelect={onOsqueryTableSelect}
          selectedOsqueryTable={selectedOsqueryTable}
          onClose={onCloseSchemaSidebar}
        />
      )}
    </div>
  );
};

export default PolicyPage;
