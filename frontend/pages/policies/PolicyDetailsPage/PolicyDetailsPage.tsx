import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";
import PATHS from "router/paths";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { IPolicy, IStoredPolicyResponse } from "interfaces/policy";
import globalPoliciesAPI from "services/entities/global_policies";
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
import Avatar from "components/Avatar";
import ShowQueryModal from "components/modals/ShowQueryModal";

interface IPolicyDetailsPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    search: string;
    query: { team_id?: string };
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
    setLastEditedQueryLabelsExcludeAny,
    setPolicyTeamId,
  } = useContext(PolicyContext);

  const {
    isRouteOk,
    teamIdForApi,
    isTeamMaintainerOrTeamAdmin,
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

  const [showQueryModal, setShowQueryModal] = useState(false);

  if (policyId === null || isNaN(policyId)) {
    router.push(PATHS.MANAGE_POLICIES);
  }

  const { isLoading, data: storedPolicy, error: apiError } = useQuery<
    IStoredPolicyResponse,
    Error,
    IPolicy
  >(["policy", policyId], () => globalPoliciesAPI.load(policyId as number), {
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
      setLastEditedQueryLabelsExcludeAny(
        returnedPolicy.labels_exclude_any || []
      );
      setPolicyTeamId(returnedPolicy.team_id ?? undefined);
    },
    onError: (error) => handlePageError(error),
  });

  useEffect(() => {
    if (storedPolicy?.name) {
      document.title = `${storedPolicy.name} | Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Policies | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, storedPolicy?.name]);

  const canEditPolicy =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainerOrTeamAdmin;

  const canRunPolicy = canEditPolicy || isObserverPlus;

  const disabledLiveQuery = config?.server_settings.live_query_disabled;

  const backToPoliciesPath = getPathWithQueryParams(PATHS.MANAGE_POLICIES, {
    team_id: teamIdForApi,
  });

  const renderAuthor = (): JSX.Element | null => {
    if (!storedPolicy) return null;
    return (
      <DataSet
        className={`${baseClass}__author`}
        title="Author"
        value={
          <>
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
          </>
        }
      />
    );
  };

  const renderPlatforms = (): JSX.Element | null => {
    if (!lastEditedQueryPlatform) return null;
    const platforms = lastEditedQueryPlatform
      .split(",")
      .map((p) => p.trim())
      .filter(Boolean);
    if (platforms.length === 0) return null;

    return (
      <DataSet
        className={`${baseClass}__platforms`}
        title="Platforms"
        value={platforms.join(", ")}
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
              <div className="name-description">
                <h1 className={`${baseClass}__policy-name`}>
                  {lastEditedQueryName}
                </h1>
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
                          getPathWithQueryParams(PATHS.EDIT_POLICY(policyId), {
                            team_id: teamIdForApi,
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
                            team_id: teamIdForApi,
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
            <PageDescription
              className={`${baseClass}__policy-description`}
              content={lastEditedQueryDescription}
            />
            {lastEditedQueryResolution && (
              <div className={`${baseClass}__resolve-section`}>
                <p className={`${baseClass}__resolve-title`}>
                  <strong>Resolve</strong>
                </p>
                <p className={`${baseClass}__resolve-text`}>
                  {lastEditedQueryResolution}
                </p>
              </div>
            )}
            {renderAuthor()}
            {currentTeam && (
              <DataSet
                className={`${baseClass}__fleet`}
                title="Fleet"
                value={currentTeam.name}
              />
            )}
            {renderPlatforms()}
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
