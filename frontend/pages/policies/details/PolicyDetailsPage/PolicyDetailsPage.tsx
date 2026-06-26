import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";
import PATHS from "router/paths";
import { AppContext } from "context/app";
import {
  IPolicy,
  IStoredPolicyResponse,
  OtherAutomationType,
} from "interfaces/policy";
import { ILabelPolicy } from "interfaces/label";
import {
  API_NO_TEAM_ID,
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";
import { PLATFORM_DISPLAY_NAMES, Platform } from "interfaces/platform";
import policiesAPI from "services/entities/policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { addGravatarUrlToResource } from "utilities/helpers";
import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import useTeamIdParam from "hooks/useTeamIdParam";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import Graphic from "components/Graphic";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Avatar from "components/Avatar";
import ShowQueryModal from "components/modals/ShowQueryModal";
import { getTicketOrWebhookInfo } from "pages/policies/helpers";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { mapAutomationRows } from "pages/policies/components";
import PolicyLabelModal, {
  IPolicyLabelModalProps,
} from "../components/PolicyLabelModal";
import PolicyAutomationsModal from "../components/PolicyAutomationsModal";
import PolicyAutomationsActivitiesTable from "../components/PolicyAutomationsActivitiesTable";

type ILabelModalData = Pick<
  IPolicyLabelModalProps,
  "includeLabels" | "includeScopeLabel" | "excludeLabels" | "excludeScopeLabel"
>;

interface IPolicyDetailsPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    search: string;
    query: { fleet_id?: string };
  };
}

const baseClass = "policy-details-page";

const getPolicyFleetName = (
  policy: IPolicy | undefined,
  teamData: ILoadTeamResponse | undefined
): string | null => {
  if (!policy) return null;
  if (policy.team_id === null) return APP_CONTEXT_ALL_TEAMS_SUMMARY.name;
  if (policy.team_id === 0) return APP_CONTEXT_NO_TEAM_SUMMARY.name;
  return teamData?.team?.name ?? null;
};

export const getLabelModalData = (policy: IPolicy): ILabelModalData => {
  let includeLabels: ILabelPolicy[] | undefined;
  let includeScopeLabel: string | undefined;
  if (policy.labels_include_any?.length) {
    includeLabels = policy.labels_include_any;
    includeScopeLabel = "have any";
  } else if (policy.labels_include_all?.length) {
    includeLabels = policy.labels_include_all;
    includeScopeLabel = "have all";
  }

  let excludeLabels: ILabelPolicy[] | undefined;
  let excludeScopeLabel: string | undefined;
  if (policy.labels_exclude_any?.length) {
    excludeLabels = policy.labels_exclude_any;
    excludeScopeLabel = "exclude any";
  } else if (policy.labels_exclude_all?.length) {
    excludeLabels = policy.labels_exclude_all;
    excludeScopeLabel = "exclude all";
  }

  return { includeLabels, includeScopeLabel, excludeLabels, excludeScopeLabel };
};

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
  } = useContext(AppContext);

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
  const [showLabelModal, setShowLabelModal] = useState(false);
  const [showAutomationsModal, setShowAutomationsModal] = useState(false);

  if (policyId === null || isNaN(policyId)) {
    router.push(PATHS.MANAGE_POLICIES);
  }

  const { isLoading, data: storedPolicy, error: apiError } = useQuery<
    IStoredPolicyResponse,
    Error,
    IPolicy
  >(["policy", policyId], () => policiesAPI.load(policyId as number), {
    enabled: isRouteOk && !!policyId,
    refetchOnWindowFocus: false,
    retry: false,
    select: (data: IStoredPolicyResponse) => data.policy,
    onError: (error) => handlePageError(error),
  });

  // Drive the team display from the policy's own team_id rather than the URL's
  // fleet_id: a user can land here from another team's list (e.g. clicking an
  // inherited policy from team 42's view), and the displayed Fleet must reflect
  // the policy's owner, not the navigation context.
  const policyTeamId = storedPolicy?.team_id;

  const { data: teamData } = useQuery<ILoadTeamResponse>(
    ["team", policyTeamId],
    () => teamsAPI.load(policyTeamId as number),
    {
      enabled: policyTeamId != null && policyTeamId >= API_NO_TEAM_ID,
      refetchOnWindowFocus: false,
    }
  );

  const policyFleetName = getPolicyFleetName(storedPolicy, teamData);

  const labelModalData = storedPolicy ? getLabelModalData(storedPolicy) : null;

  const {
    state: ticketOrWebhookState,
    policyIds: currentAutomatedPolicies,
  } = getTicketOrWebhookInfo(
    storedPolicy?.team_id == null ? config ?? undefined : teamData?.team
  );
  const otherAutomationType: OtherAutomationType | undefined =
    ticketOrWebhookState === "disabled" ? undefined : ticketOrWebhookState;

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

  const canEditLabels =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isGlobalTechnician ||
    isTeamMaintainerOrTeamAdmin ||
    isTeamTechnician;

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
    if (!storedPolicy?.platform) return null;
    const platforms = storedPolicy.platform
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

  const openLabelModal = () => setShowLabelModal(true);

  const renderLabels = (): JSX.Element | null => {
    if (!labelModalData) return null;

    const { includeLabels, excludeLabels } = labelModalData;
    const allLabels = [...(includeLabels ?? []), ...(excludeLabels ?? [])];
    if (!allLabels.length) return null;

    const firstLabel = allLabels[0];
    const moreLabels = allLabels.length - 1;
    return (
      <DataSet
        className={`${baseClass}__labels`}
        title="Labels"
        value={
          <Button variant="link" onClick={openLabelModal}>
            {firstLabel.name}
            {moreLabels > 0 && ` + ${moreLabels} more`}
          </Button>
        }
      />
    );
  };

  const renderFleetName = () => {
    if (!policyFleetName) return null;
    return (
      <DataSet
        className={`${baseClass}__fleet`}
        title="Fleet"
        value={policyFleetName}
      />
    );
  };

  const renderResolution = () => {
    if (!storedPolicy?.resolution) return null;
    return (
      <DataSet
        className={`${baseClass}__resolve`}
        title="Resolve"
        value={storedPolicy.resolution}
        multiline
      />
    );
  };

  const openAutomationsModal = () => setShowAutomationsModal(true);

  const renderAutomations = () => {
    const emptyState = (
      <DataSet
        className={`${baseClass}__automations`}
        title="Automations"
        value="---"
      />
    );

    if (!storedPolicy) return emptyState;

    const automations = mapAutomationRows(
      storedPolicy,
      currentAutomatedPolicies,
      otherAutomationType
    );
    if (!automations.length) return emptyState;

    const firstAutomation = automations[0];
    const moreCount = automations.length - 1;
    return (
      <DataSet
        className={`${baseClass}__automations`}
        title="Automations"
        value={
          <Button variant="link" onClick={openAutomationsModal}>
            {firstAutomation.isSoftware ? (
              <SoftwareIcon
                name={firstAutomation.iconName ?? firstAutomation.name}
                url={firstAutomation.iconUrl}
                size="small"
              />
            ) : (
              firstAutomation.graphicName && (
                <Graphic
                  name={firstAutomation.graphicName}
                  className={
                    firstAutomation.graphicName === "file-sh" ||
                    firstAutomation.graphicName === "file-ps1"
                      ? "scale-40-24"
                      : ""
                  }
                />
              )
            )}
            {firstAutomation.name}
            {moreCount > 0 && ` + ${moreCount} more`}
          </Button>
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
                  <TooltipTruncatedText
                    value={storedPolicy?.name}
                    fixedPositionStrategy
                  />
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
                  content={storedPolicy?.description}
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
            <div className={`${baseClass}__details`}>
              <div className={`${baseClass}__properties`}>
                {renderFleetName()}
                {renderPlatforms()}
                {renderLabels()}
                {renderAutomations()}
                {renderAuthor()}
              </div>
              {renderResolution()}
            </div>
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
      {!isLoading && !apiError && storedPolicy && (
        <PolicyAutomationsActivitiesTable
          policy={storedPolicy}
          currentAutomatedPolicies={currentAutomatedPolicies}
          otherAutomationType={otherAutomationType}
          canResetPolicy={canEditPolicy}
        />
      )}
      {showQueryModal && (
        <ShowQueryModal
          query={storedPolicy?.query}
          onCancel={() => setShowQueryModal(false)}
        />
      )}
      {showLabelModal && labelModalData && (
        <PolicyLabelModal
          includeLabels={labelModalData.includeLabels}
          includeScopeLabel={labelModalData.includeScopeLabel}
          excludeLabels={labelModalData.excludeLabels}
          excludeScopeLabel={labelModalData.excludeScopeLabel}
          onLabelClick={
            canEditLabels
              ? (labelId) => router.push(PATHS.LABEL_EDIT(labelId))
              : undefined
          }
          onClose={() => setShowLabelModal(false)}
        />
      )}
      {showAutomationsModal && storedPolicy && (
        <PolicyAutomationsModal
          storedPolicy={storedPolicy}
          currentAutomatedPolicies={currentAutomatedPolicies}
          otherAutomationType={otherAutomationType}
          onClose={() => setShowAutomationsModal(false)}
        />
      )}
    </MainContent>
  );
};

export default PolicyDetailsPage;
