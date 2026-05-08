import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";
import { noop } from "lodash";
import PATHS from "router/paths";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { IPolicy, IStoredPolicyResponse } from "interfaces/policy";
import { ILabelPolicy } from "interfaces/label";
import { API_ALL_TEAMS_ID, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { PLATFORM_DISPLAY_NAMES, Platform } from "interfaces/platform";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { addGravatarUrlToResource } from "utilities/helpers";
import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import useTeamIdParam from "hooks/useTeamIdParam";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import Avatar from "components/Avatar";
import ShowQueryModal from "components/modals/ShowQueryModal";
import PolicyAutomations from "pages/policies/edit/components/PolicyAutomations";

interface IPolicyDetailsPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    search: string;
    query: { fleet_id?: string; inherited_policy?: string };
  };
}

const baseClass = "policy-details-page";

const PolicyDetailsPage = ({
  router,
  params: { id: paramsPolicyId },
  location,
}: IPolicyDetailsPageProps): JSX.Element => {
  const policyId = paramsPolicyId ? parseInt(paramsPolicyId, 10) : null;
  const handlePageError = useErrorHandler();

  const {
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalTechnician,
    isOnGlobalTeam,
    config,
    currentTeam,
  } = useContext(AppContext);

  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryResolution,
    lastEditedQueryBody,
    lastEditedQueryPlatform,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
    setLastEditedQueryLabelsIncludeAny,
    setLastEditedQueryLabelsIncludeAll,
    setLastEditedQueryLabelsExcludeAny,
    setPolicyTeamId,
  } = useContext(PolicyContext);

  const {
    isRouteOk,
    teamIdForApi,
    isTeamMaintainerOrTeamAdmin,
    isTeamTechnician,
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

  const [showQueryModal, setShowQueryModal] = useState(false);

  if (policyId === null || isNaN(policyId)) {
    router.push(PATHS.MANAGE_POLICIES);
  }

  // Inherited (global) policies clicked from a team's policy list pass this
  // hint so we fetch from the global endpoint even though the URL carries a
  // fleet_id (team users can't render a teamless URL — useTeamIdParam
  // redirects them to their default team).
  const isInheritedPolicyLink = location.query.inherited_policy === "true";

  const { isLoading, data: storedPolicy, error: apiError } = useQuery<
    IStoredPolicyResponse,
    Error,
    IPolicy
  >(
    ["policy", policyId, teamIdForApi, isInheritedPolicyLink],
    () =>
      !isInheritedPolicyLink && teamIdForApi !== undefined
        ? teamPoliciesAPI.load(teamIdForApi, policyId as number)
        : globalPoliciesAPI.load(policyId as number),
    {
      enabled: isRouteOk && !!policyId,
      refetchOnWindowFocus: false,
      retry: false,
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
        setLastEditedQueryLabelsIncludeAll(
          returnedPolicy.labels_include_all || []
        );
        setLastEditedQueryLabelsExcludeAny(
          returnedPolicy.labels_exclude_any || []
        );
        const deNulledTeamId = returnedPolicy.team_id ?? undefined;
        setPolicyTeamId(
          deNulledTeamId === API_ALL_TEAMS_ID
            ? APP_CONTEXT_ALL_TEAMS_ID
            : deNulledTeamId
        );
      },
      onError: (error) => handlePageError(error),
    }
  );

  const { data: teamData } = useQuery<ILoadTeamResponse>(
    ["team", teamIdForApi],
    () => teamsAPI.load(teamIdForApi as number),
    {
      enabled: !!teamIdForApi && teamIdForApi > 0,
      refetchOnWindowFocus: false,
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

  useEffect(() => {
    if (storedPolicy?.name) {
      document.title = `${storedPolicy.name} | Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, storedPolicy?.name]);

  const isInheritedPolicy = storedPolicy?.team_id === null;

  const canEditPolicy =
    (isGlobalAdmin || isGlobalMaintainer || isTeamMaintainerOrTeamAdmin) &&
    // Team users cannot edit inherited (global) policies
    !(isInheritedPolicy && !isOnGlobalTeam);

  const canRunPolicy =
    isObserverPlus ||
    isTeamMaintainerOrTeamAdmin ||
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isGlobalTechnician ||
    isTeamTechnician;

  const disabledLiveQuery = config?.server_settings.live_query_disabled;

  const backToPoliciesPath = getPathWithQueryParams(PATHS.MANAGE_POLICIES, {
    fleet_id: teamIdForApi,
  });

  const renderAuthor = (): JSX.Element | null => {
    if (!storedPolicy) return null;
    return (
      <DataSet
        className={`${baseClass}__author`}
        title="Author"
        value={
          <div className={`${baseClass}__author-info`}>
            <Avatar
              user={addGravatarUrlToResource({
                email: storedPolicy.author_email,
              })}
              size="xsmall"
            />
            <span>
              {storedPolicy.author_name === currentUser?.name
                ? "You"
                : storedPolicy.author_name}
            </span>
          </div>
        }
      />
    );
  };

  const renderPlatforms = (): JSX.Element | null => {
    if (!lastEditedQueryPlatform) return null;
    const platforms = lastEditedQueryPlatform
      .split(",")
      .map((p) => p.trim())
      .filter((p): p is Platform => p in PLATFORM_DISPLAY_NAMES);
    if (platforms.length === 0) return null;

    return (
      <DataSet
        className={`${baseClass}__platforms`}
        title="Platforms"
        value={
          <div className={`${baseClass}__platform-list`}>
            {platforms.map((platform) => (
              <span key={platform} className={`${baseClass}__platform-item`}>
                <Icon name={platform} color="ui-fleet-black-75" />
                {PLATFORM_DISPLAY_NAMES[platform] || platform}
              </span>
            ))}
          </div>
        }
      />
    );
  };

  const onLabelClick = (label: ILabelPolicy) => {
    router.push(PATHS.MANAGE_HOSTS_LABEL(label.id));
  };

  const renderLabels = (): JSX.Element | null => {
    const includeAny = storedPolicy?.labels_include_any;
    const includeAll = storedPolicy?.labels_include_all;
    const excludeAny = storedPolicy?.labels_exclude_any;

    let labels: ILabelPolicy[] | undefined;
    let scopeLabel: string;
    if (includeAny?.length) {
      labels = includeAny;
      scopeLabel = "have any";
    } else if (includeAll?.length) {
      labels = includeAll;
      scopeLabel = "have all";
    } else if (excludeAny?.length) {
      labels = excludeAny;
      scopeLabel = "exclude any";
    } else {
      return null;
    }

    return (
      <DataSet
        className={`${baseClass}__labels`}
        title="Labels"
        value={
          <div className={`${baseClass}__labels-section`}>
            <p>
              Policy will target hosts that <b>{scopeLabel}</b> of these labels:
            </p>
            <ul className={`${baseClass}__labels-list`}>
              {labels?.map((label: ILabelPolicy) => (
                <li key={label.id}>
                  <Button
                    onClick={() => onLabelClick(label)}
                    variant="grey-pill"
                    className={`${baseClass}__label-pill`}
                  >
                    {label.name}
                  </Button>
                </li>
              ))}
            </ul>
          </div>
        }
      />
    );
  };

  const renderHeader = () => {
    return (
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton text="Back to policies" path={backToPoliciesPath} />
        </div>
        {!isLoading && !apiError && (
          <>
            <div className={`${baseClass}__title-bar`}>
              <div className={`${baseClass}__name-description`}>
                <h1 className={`${baseClass}__policy-name`}>
                  {lastEditedQueryName}
                  {storedPolicy?.critical && (
                    <TooltipWrapper
                      tipContent="This policy has been marked as critical."
                      showArrow
                      underline={false}
                    >
                      <Icon
                        className="critical-policy-icon"
                        name="policy"
                        color="ui-fleet-black-50"
                      />
                    </TooltipWrapper>
                  )}
                </h1>
                <PageDescription
                  className={`${baseClass}__policy-description`}
                  content={lastEditedQueryDescription}
                />
              </div>
              <div className={`${baseClass}__action-button-container`}>
                <Button
                  className={`${baseClass}__show-query-btn`}
                  onClick={() => setShowQueryModal(true)}
                  variant="inverse"
                >
                  Show query
                </Button>
                {canRunPolicy && (
                  <Button
                    className={`${baseClass}__run`}
                    variant="inverse"
                    onClick={() => {
                      policyId &&
                        router.push(
                          getPathWithQueryParams(PATHS.LIVE_POLICY(policyId), {
                            fleet_id: teamIdForApi,
                          })
                        );
                    }}
                    disabled={!!disabledLiveQuery}
                  >
                    Run policy <Icon name="run" />
                  </Button>
                )}
                {canEditPolicy && (
                  <Button
                    onClick={() => {
                      policyId &&
                        router.push(
                          getPathWithQueryParams(PATHS.EDIT_POLICY(policyId), {
                            fleet_id: teamIdForApi,
                          })
                        );
                    }}
                    className={`${baseClass}__edit-policy-btn`}
                  >
                    Edit policy
                  </Button>
                )}
              </div>
            </div>
            {lastEditedQueryResolution && (
              <DataSet
                className={`${baseClass}__resolve`}
                title="Resolve"
                value={lastEditedQueryResolution}
              />
            )}
            {renderAuthor()}
            {storedPolicy && (
              <DataSet
                className={`${baseClass}__fleet`}
                title="Fleet"
                value={
                  storedPolicy.team_id === null
                    ? "All fleets"
                    : currentTeam?.name
                }
              />
            )}
            {renderPlatforms()}
            {renderLabels()}
            {storedPolicy && (
              <PolicyAutomations
                storedPolicy={storedPolicy}
                currentAutomatedPolicies={currentAutomatedPolicies}
                onAddAutomation={noop}
                isAddingAutomation={false}
              />
            )}
          </>
        )}
      </>
    );
  };

  if (!isRouteOk) {
    return <Spinner />;
  }

  return (
    <MainContent className={baseClass}>
      {isLoading ? <Spinner /> : renderHeader()}
      {showQueryModal && (
        <ShowQueryModal
          query={lastEditedQueryBody}
          onCancel={() => setShowQueryModal(false)}
        />
      )}
    </MainContent>
  );
};

export default PolicyDetailsPage;
