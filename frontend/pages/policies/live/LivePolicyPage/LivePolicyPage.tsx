import React, { useState, useEffect, useContext, useCallback } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter, Params } from "react-router/lib/Router";
import PATHS from "router/paths";
import useTeamIdParam from "hooks/useTeamIdParam";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { LIVE_QUERY_STEPS, DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import policiesAPI from "services/entities/policies";
import hostAPI from "services/entities/hosts";
import { IHost, IHostResponse } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IPolicy, IStoredPolicyResponse } from "interfaces/policy";
import { ITarget } from "interfaces/target";

import MainContent from "components/MainContent";
import SelectTargets from "components/LiveQuery/SelectTargets";

import RunQuery from "pages/policies/live/screens/RunQuery";

interface ILivePolicyPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    query: { host_ids?: string; fleet_id?: string };
    search: string;
  };
}

const baseClass = "live-policy-page";

const LivePolicyPage = ({
  router,
  params: { id: paramsPolicyId },
  location,
}: ILivePolicyPageProps): JSX.Element => {
  const policyId = paramsPolicyId ? parseInt(paramsPolicyId, 10) : null;
  const handlePageError = useErrorHandler();

  const { currentTeamId } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
  });

  const { config } = useContext(AppContext);
  const {
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
    setLastEditedQueryLabelsIncludeAny,
    setLastEditedQueryLabelsExcludeAny,
  } = useContext(PolicyContext);

  const [queryParamHostsAdded, setQueryParamHostsAdded] = useState(false);
  const [step, setStep] = useState(LIVE_QUERY_STEPS[1]);
  const [selectedTargets, setSelectedTargets] = useState<ITarget[]>([]);
  const [targetedHosts, setTargetedHosts] = useState<IHost[]>([]);
  const [targetedLabels, setTargetedLabels] = useState<ILabel[]>([]);
  const [targetedTeams, setTargetedTeams] = useState<ITeam[]>([]);
  const [targetsTotalCount, setTargetsTotalCount] = useState(0);

  const disabledLiveQuery = config?.server_settings.live_query_disabled;
  const teamIdForApi = currentTeamId === -1 ? undefined : currentTeamId;

  // Reroute users out of live flow when live queries are globally disabled
  // Reroute users out of live flow when live queries are globally disabled
  useEffect(() => {
    if (disabledLiveQuery) {
      const path = policyId
        ? PATHS.POLICY_DETAILS(policyId)
        : PATHS.MANAGE_POLICIES;

      router.push(getPathWithQueryParams(path, { fleet_id: teamIdForApi }));
    }
  }, [disabledLiveQuery, policyId, router, teamIdForApi]);

  const { data: storedPolicy } = useQuery<
    IStoredPolicyResponse,
    Error,
    IPolicy
  >(
    ["policy", policyId, teamIdForApi],
    () => policiesAPI.load(policyId as number),
    {
      enabled: !!policyId,
      refetchOnWindowFocus: false,
      select: (data: IStoredPolicyResponse) => data.policy,
      onSuccess: (returnedPolicy) => {
        setLastEditedQueryId(returnedPolicy.id);
        setLastEditedQueryName(returnedPolicy.name);
        setLastEditedQueryDescription(returnedPolicy.description);
        setLastEditedQueryBody(returnedPolicy.query);
        setLastEditedQueryResolution(returnedPolicy.resolution);
        setLastEditedQueryCritical(returnedPolicy.critical);
        setLastEditedQueryPlatform(returnedPolicy.platform);
        setLastEditedQueryLabelsIncludeAny(
          returnedPolicy.labels_include_any || []
        );
        setLastEditedQueryLabelsExcludeAny(
          returnedPolicy.labels_exclude_any || []
        );
      },
      onError: (error) => handlePageError(error),
    }
  );

  const hostIdFromURL = location.query.host_ids
    ? parseInt(location.query.host_ids as string, 10)
    : null;

  useQuery<IHostResponse, Error, IHost>(
    ["hostFromURL", hostIdFromURL, teamIdForApi],
    () => hostAPI.loadHostDetails(hostIdFromURL as number),
    {
      enabled: !!hostIdFromURL && !queryParamHostsAdded,
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

  // Updates title that shows up on browser tabs
  useEffect(() => {
    if (storedPolicy?.name) {
      document.title = `Run ${storedPolicy.name} | Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, storedPolicy?.name]);

  const goToQueryEditor = useCallback(() => {
    const path = policyId ? PATHS.EDIT_POLICY(policyId) : PATHS.NEW_POLICY;

    router.push(getPathWithQueryParams(path, { fleet_id: teamIdForApi }));
  }, [policyId, router, teamIdForApi]);

  const renderScreen = () => {
    const step1Props = {
      baseClass,
      selectedTargets,
      targetedHosts,
      targetedLabels,
      targetedTeams,
      targetsTotalCount,
      goToQueryEditor,
      goToRunQuery: () => setStep(LIVE_QUERY_STEPS[2]),
      setSelectedTargets,
      setTargetedHosts,
      setTargetedLabels,
      setTargetedTeams,
      setTargetsTotalCount,
      isLivePolicy: true,
    };

    const step2Props = {
      selectedTargets,
      storedPolicy,
      setSelectedTargets,
      goToQueryEditor,
      targetsTotalCount,
    };

    switch (step) {
      case LIVE_QUERY_STEPS[2]:
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

export default LivePolicyPage;
