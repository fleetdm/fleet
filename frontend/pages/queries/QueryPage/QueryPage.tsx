import React, { useState, useEffect, useContext, useCallback } from "react";
import { useQuery, useMutation } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter, Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { QUERIES_PAGE_STEPS, DEFAULT_QUERY } from "utilities/constants";
import queryAPI from "services/entities/queries";
import hostAPI from "services/entities/hosts";
import statusAPI from "services/entities/status";
import { IHost, IHostResponse } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IQueryFormData, IQuery, IStoredQueryResponse } from "interfaces/query";
import { ITarget } from "interfaces/target";

import QuerySidePanel from "components/side_panels/QuerySidePanel";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import SelectTargets from "components/LiveQuery/SelectTargets";
import CustomLink from "components/CustomLink";

import QueryEditor from "pages/queries/QueryPage/screens/QueryEditor";
import RunQuery from "pages/queries/QueryPage/screens/RunQuery";

interface IQueryPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    query: { host_ids: string };
  };
}

const baseClass = "query-page";

const QueryPage = ({
  router,
  params: { id: paramsQueryId },
  location: { query: URLQuerySearch },
}: IQueryPageProps): JSX.Element => {
  const queryId = paramsQueryId ? parseInt(paramsQueryId, 10) : null;

  const handlePageError = useErrorHandler();
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    isObserverPlus,
    isAnyTeamObserverPlus,
  } = useContext(AppContext);
  const {
    selectedOsqueryTable,
    setSelectedOsqueryTable,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
  } = useContext(QueryContext);

  const [queryParamHostsAdded, setQueryParamHostsAdded] = useState(false);
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

  // disabled on page load so we can control the number of renders
  // else it will re-populate the context on occasion
  const {
    isLoading: isStoredQueryLoading,
    data: storedQuery,
    error: storedQueryError,
  } = useQuery<IStoredQueryResponse, Error, IQuery>(
    ["query", queryId],
    () => queryAPI.load(queryId as number),
    {
      enabled: !!queryId,
      refetchOnWindowFocus: false,
      select: (data: IStoredQueryResponse) => data.query,
      onSuccess: (returnedQuery) => {
        setLastEditedQueryId(returnedQuery.id);
        setLastEditedQueryName(returnedQuery.name);
        setLastEditedQueryDescription(returnedQuery.description);
        setLastEditedQueryBody(returnedQuery.query);
        setLastEditedQueryObserverCanRun(returnedQuery.observer_can_run);
      },
      onError: (error) => handlePageError(error),
    }
  );

  useQuery<IHostResponse, Error, IHost>(
    "hostFromURL",
    () =>
      hostAPI.loadHostDetails(parseInt(URLQuerySearch.host_ids as string, 10)),
    {
      enabled: !!URLQuerySearch.host_ids && !queryParamHostsAdded,
      select: (data: IHostResponse) => data.host,
      onSuccess: (host) => {
        setTargetedHosts((prevHosts) =>
          prevHosts.filter((h) => h.id !== host.id).concat(host)
        );
        const targets = selectedTargets;
        host.target_type = "hosts";
        targets.push(host);
        setSelectedTargets([...targets]);
        if (!queryParamHostsAdded) {
          setQueryParamHostsAdded(true);
        }
        router.replace(location.pathname);
      },
    }
  );

  const { mutateAsync: createQuery } = useMutation((formData: IQueryFormData) =>
    queryAPI.create(formData)
  );

  const detectIsFleetQueryRunnable = () => {
    statusAPI.live_query().catch(() => {
      setIsLiveQueryRunnable(false);
    });
  };

  useEffect(() => {
    detectIsFleetQueryRunnable();
    setLastEditedQueryId(DEFAULT_QUERY.id);
    setLastEditedQueryName(DEFAULT_QUERY.name);
    setLastEditedQueryDescription(DEFAULT_QUERY.description);
    setLastEditedQueryBody(DEFAULT_QUERY.query);
    setLastEditedQueryObserverCanRun(DEFAULT_QUERY.observer_can_run);
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

  const goToQueryEditor = useCallback(() => setStep(QUERIES_PAGE_STEPS[1]), []);

  const renderScreen = () => {
    const step1Opts = {
      router,
      baseClass,
      queryIdForEdit: queryId,
      showOpenSchemaActionText,
      storedQuery,
      isStoredQueryLoading,
      storedQueryError,
      createQuery,
      onOsqueryTableSelect,
      goToSelectTargets: () => setStep(QUERIES_PAGE_STEPS[2]),
      onOpenSchemaSidebar,
      renderLiveQueryWarning,
    };

    const step2Opts = {
      baseClass,
      queryId,
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
      queryId,
      selectedTargets,
      storedQuery,
      setSelectedTargets,
      goToQueryEditor,
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
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isAnyTeamMaintainerOrTeamAdmin ||
      isObserverPlus ||
      isAnyTeamObserverPlus);

  return (
    <>
      <MainContent className={baseClass}>
        <div className={`${baseClass}_wrapper`}>{renderScreen()}</div>
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

export default QueryPage;
