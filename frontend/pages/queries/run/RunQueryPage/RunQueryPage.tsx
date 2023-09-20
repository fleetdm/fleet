import React, { useState, useEffect, useContext, useCallback } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter, Params } from "react-router/lib/Router";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { RUN_QUERY_STEPS, DEFAULT_QUERY } from "utilities/constants";
import queryAPI from "services/entities/queries";
import hostAPI from "services/entities/hosts";
import statusAPI from "services/entities/status";
import { IHost, IHostResponse } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import MainContent from "components/MainContent";
import SelectTargets from "components/LiveQuery/SelectTargets";

import RunQuery from "pages/queries/run/screens/RunQuery";
import useTeamIdParam from "hooks/useTeamIdParam";

interface IRunQueryPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    query: { host_ids: string; team_id?: string };
    search: string;
  };
}

const baseClass = "run-query-page";

const RunQueryPage = ({
  router,
  params: { id: paramsQueryId },
  location,
}: IRunQueryPageProps): JSX.Element => {
  const queryId = paramsQueryId ? parseInt(paramsQueryId, 10) : null;
  const {
    currentTeamName: teamNameForQuery,
    teamIdForApi: apiTeamIdForQuery,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const handlePageError = useErrorHandler();
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    isObserverPlus,
    isAnyTeamObserverPlus,
  } = useContext(AppContext);
  const {
    selectedQueryTargets,
    setSelectedQueryTargets,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
    setLastEditedQueryFrequency,
    setLastEditedQueryLoggingType,
    setLastEditedQueryMinOsqueryVersion,
    setLastEditedQueryPlatforms,
  } = useContext(QueryContext);

  const [queryParamHostsAdded, setQueryParamHostsAdded] = useState(false);
  const [step, setStep] = useState(RUN_QUERY_STEPS[1]);
  const [targetedHosts, setTargetedHosts] = useState<IHost[]>([]);
  const [targetedLabels, setTargetedLabels] = useState<ILabel[]>([]);
  const [targetedTeams, setTargetedTeams] = useState<ITeam[]>([]);
  const [targetsTotalCount, setTargetsTotalCount] = useState(0);
  const [isLiveQueryRunnable, setIsLiveQueryRunnable] = useState(true);

  const TAGGED_TEMPLATES = {
    queryByHostRoute: (hostId: number | undefined | null) => {
      return `${hostId ? `?host_ids=${hostId}` : ""}`;
    },
  };

  // disabled on page load so we can control the number of renders
  // else it will re-populate the context on occasion
  const { data: storedQuery } = useQuery<
    IGetQueryResponse,
    Error,
    ISchedulableQuery
  >(["query", queryId], () => queryAPI.load(queryId as number), {
    enabled: !!queryId,
    refetchOnWindowFocus: false,
    select: (data) => data.query,
    onSuccess: (returnedQuery) => {
      setLastEditedQueryId(returnedQuery.id);
      setLastEditedQueryName(returnedQuery.name);
      setLastEditedQueryDescription(returnedQuery.description);
      setLastEditedQueryBody(returnedQuery.query);
      setLastEditedQueryObserverCanRun(returnedQuery.observer_can_run);
      setLastEditedQueryFrequency(returnedQuery.interval);
      setLastEditedQueryPlatforms(returnedQuery.platform);
      setLastEditedQueryLoggingType(returnedQuery.logging);
      setLastEditedQueryMinOsqueryVersion(returnedQuery.min_osquery_version);
    },
    onError: (error) => handlePageError(error),
  });

  useQuery<IHostResponse, Error, IHost>(
    "hostFromURL",
    () =>
      hostAPI.loadHostDetails(parseInt(location.query.host_ids as string, 10)),
    {
      enabled: !!location.query.host_ids && !queryParamHostsAdded,
      select: (data: IHostResponse) => data.host,
      onSuccess: (host) => {
        setTargetedHosts((prevHosts) =>
          prevHosts.filter((h) => h.id !== host.id).concat(host)
        );
        const targets = selectedQueryTargets;
        host.target_type = "hosts";
        targets.push(host);
        setSelectedQueryTargets([...targets]);
        if (!queryParamHostsAdded) {
          setQueryParamHostsAdded(true);
        }
        router.replace(location.pathname);
      },
    }
  );

  const detectIsFleetQueryRunnable = () => {
    statusAPI.live_query().catch(() => {
      setIsLiveQueryRunnable(false);
    });
  };

  useEffect(() => {
    detectIsFleetQueryRunnable();
    if (!queryId) {
      setLastEditedQueryId(DEFAULT_QUERY.id);
      setLastEditedQueryName(DEFAULT_QUERY.name);
      setLastEditedQueryDescription(DEFAULT_QUERY.description);
      setLastEditedQueryBody(DEFAULT_QUERY.query);
      setLastEditedQueryObserverCanRun(DEFAULT_QUERY.observer_can_run);
      setLastEditedQueryFrequency(DEFAULT_QUERY.interval);
      setLastEditedQueryLoggingType(DEFAULT_QUERY.logging);
      setLastEditedQueryMinOsqueryVersion(DEFAULT_QUERY.min_osquery_version);
      setLastEditedQueryPlatforms(DEFAULT_QUERY.platform);
    }
  }, [queryId]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Run live query | Discover TLS certificates | Fleet for osquery
    document.title = `Run live query | ${storedQuery?.name} | Fleet for osquery`;
  }, [location.pathname, storedQuery?.name]);

  const goToQueryEditor = useCallback(
    () => queryId && router.push(PATHS.EDIT_QUERY(queryId)),
    []
  );
  // const params = { id: paramsQueryId };

  const renderScreen = () => {
    const step1Props = {
      baseClass,
      queryId,
      selectedTargets: selectedQueryTargets,
      targetedHosts,
      targetedLabels,
      targetedTeams,
      targetsTotalCount,
      goToQueryEditor,
      goToRunQuery: () => setStep(RUN_QUERY_STEPS[2]),
      setSelectedTargets: setSelectedQueryTargets,
      setTargetedHosts,
      setTargetedLabels,
      setTargetedTeams,
      setTargetsTotalCount,
    };

    const step2Props = {
      queryId,
      selectedTargets: selectedQueryTargets,
      storedQuery,
      setSelectedTargets: setSelectedQueryTargets,
      goToQueryEditor,
      targetsTotalCount,
    };

    switch (step) {
      case RUN_QUERY_STEPS[2]:
        return <RunQuery {...step2Props} />;
      default:
        return <SelectTargets {...step1Props} />;
    }
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}_wrapper`}>{renderScreen()}</div>
    </MainContent>
  );
};

export default RunQueryPage;
