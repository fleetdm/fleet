import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import { get, has, noop } from "lodash";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";

import { IAutomationsConfig, IConfig } from "interfaces/config";
import { IPolicyStats, ILoadAllPoliciesResponse } from "interfaces/policy";
import { ITeamAutomationsConfig } from "interfaces/team";
import { IWebhookFailingPolicies } from "interfaces/webhook";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import TableDataError from "components/DataError";
import PoliciesListWrapper from "./components/PoliciesListWrapper";
import ManageAutomationsModal from "./components/ManageAutomationsModal";
import AddPolicyModal from "./components/AddPolicyModal";
import RemovePoliciesModal from "./components/RemovePoliciesModal";

interface IManagePoliciesPageProps {
  router: InjectedRouter;
  location: {
    action: string;
    hash: string;
    key: string;
    pathname: string;
    query: { team_id?: string };
    search: string;
  };
}

const baseClass = "manage-policies-page";

const ManagePolicyPage = ({
  router,
  location,
}: IManagePoliciesPageProps): JSX.Element => {
  const {
    availableTeams,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isFreeTier,
    isPremiumTier,
    isTeamAdmin,
    isTeamMaintainer,
    currentTeam,
    setCurrentTeam,
    setConfig,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const teamId = location?.query?.team_id
    ? parseInt(location?.query?.team_id, 10)
    : 0;

  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const { setResetSelectedRows } = useContext(TableContext);

  const [isAutomationsLoading, setIsAutomationsLoading] = useState<boolean>(
    false
  );
  const [isRemovingPolicy, setIsRemovingPolicy] = useState<boolean>(false);
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<number[]>([]);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showRemovePoliciesModal, setShowRemovePoliciesModal] = useState(false);
  const [showInheritedPolicies, setShowInheritedPolicies] = useState(false);
  const [currentAutomatedPolicies, setCurrentAutomatedPolicies] = useState<
    number[]
  >();

  useEffect(() => {
    setLastEditedQueryPlatform(null);
  }, []);

  const {
    data: globalPolicies,
    error: globalPoliciesError,
    isFetching: isFetchingGlobalPolicies,
    isStale: isStaleGlobalPolicies,
    refetch: refetchGlobalPolicies,
  } = useQuery<ILoadAllPoliciesResponse, Error, IPolicyStats[]>(
    ["globalPolicies"],
    () => {
      return globalPoliciesAPI.loadAll();
    },
    {
      enabled: !!availableTeams,
      select: (data) => data.policies,
      staleTime: 3000,
    }
  );

  const {
    data: teamPolicies,
    error: teamPoliciesError,
    isFetching: isFetchingTeamPolicies,
    refetch: refetchTeamPolicies,
  } = useQuery<ILoadAllPoliciesResponse, Error, IPolicyStats[]>(
    ["teamPolicies", teamId],
    () => teamPoliciesAPI.loadAll(teamId),
    {
      enabled: !!availableTeams && isPremiumTier && !!teamId,
      select: (data) => data.policies,
    }
  );

  const canAddOrRemovePolicy =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;
  const canManageAutomations = isGlobalAdmin || isTeamAdmin;

  const {
    data: automationsConfig,
    isFetching: isFetchingAutomationsConfig,
    refetch: refetchAutomationsConfig,
  } = useQuery<
    IConfig | ILoadTeamResponse,
    Error,
    IAutomationsConfig | ITeamAutomationsConfig
  >(
    ["automationsConfig", teamId],
    () => {
      return teamId ? teamsAPI.load(teamId) : configAPI.loadAll();
    },
    {
      enabled: canAddOrRemovePolicy,
      select: (data) => {
        if (has(data, "team")) {
          return get(data, "team");
        }
        return data;
      },
      onSuccess: (data) => {
        setCurrentAutomatedPolicies(
          data.webhook_settings?.failing_policies_webhook?.policy_ids
        );

        if (has(data, "org_info")) {
          setConfig(data as IConfig);
        }
      },
    }
  );

  const refetchPolicies = (id?: number) => {
    refetchGlobalPolicies();
    if (id) {
      refetchTeamPolicies();
    }
  };

  const findAvailableTeam = (id: number) => {
    return availableTeams?.find((t) => t.id === id);
  };

  const handleTeamSelect = (id: number) => {
    const { MANAGE_POLICIES } = PATHS;

    const selectedTeam = findAvailableTeam(id);
    const path = selectedTeam?.id
      ? `${MANAGE_POLICIES}?team_id=${selectedTeam.id}`
      : MANAGE_POLICIES;

    router.replace(path);
    setShowInheritedPolicies(false);
    setSelectedPolicyIds([]);
    setCurrentTeam(selectedTeam);
    isStaleGlobalPolicies && refetchGlobalPolicies();
  };

  const toggleManageAutomationsModal = () =>
    setShowManageAutomationsModal(!showManageAutomationsModal);

  const togglePreviewPayloadModal = useCallback(() => {
    setShowPreviewPayloadModal(!showPreviewPayloadModal);
  }, [setShowPreviewPayloadModal, showPreviewPayloadModal]);

  const toggleAddPolicyModal = () => setShowAddPolicyModal(!showAddPolicyModal);

  const toggleRemovePoliciesModal = () =>
    setShowRemovePoliciesModal(!showRemovePoliciesModal);

  const toggleShowInheritedPolicies = () =>
    setShowInheritedPolicies(!showInheritedPolicies);

  const handleUpdateAutomations = async (
    formData: IAutomationsConfig | ITeamAutomationsConfig
  ) => {
    setIsAutomationsLoading(true);
    try {
      await (teamId
        ? teamsAPI.update(formData, teamId)
        : configAPI.update(formData));
      renderFlash("success", "Successfully updated policy automations.");
    } catch {
      renderFlash(
        "error",
        "Could not update policy automations. Please try again."
      );
    } finally {
      toggleManageAutomationsModal();
      setIsAutomationsLoading(false);
      refetchAutomationsConfig();
    }
  };

  const onCreateWebhookSubmit = async ({
    destination_url,
    policy_ids,
    enable_failing_policies_webhook,
  }: IWebhookFailingPolicies) => {
    setIsAutomationsLoading(true);
    try {
      const api = teamId ? teamsAPI : configAPI;
      const secondParam = teamId || undefined;
      const data = {
        webhook_settings: {
          failing_policies_webhook: {
            destination_url,
            policy_ids,
            enable_failing_policies_webhook,
          },
        },
      };
      setIsAutomationsLoading(true);

      const request = api.update(data, secondParam);
      await request.then(() => {
        renderFlash("success", "Successfully updated policy automations.");
      });
    } catch {
      renderFlash(
        "error",
        "Could not update policy automations. Please try again."
      );
    } finally {
      toggleManageAutomationsModal();
      setIsAutomationsLoading(false);
      refetchAutomationsConfig();
    }
  };

  const onAddPolicyClick = () => {
    setLastEditedQueryName("");
    setLastEditedQueryDescription("");
    setLastEditedQueryResolution("");
    toggleAddPolicyModal();
  };

  const onRemovePoliciesClick = (selectedTableIds: number[]): void => {
    toggleRemovePoliciesModal();
    setSelectedPolicyIds(selectedTableIds);
  };

  const onRemovePoliciesSubmit = async () => {
    const id = currentTeam?.id;
    setIsRemovingPolicy(true);
    try {
      const request = id
        ? teamPoliciesAPI.destroy(id, selectedPolicyIds)
        : globalPoliciesAPI.destroy(selectedPolicyIds);

      await request.then(() => {
        renderFlash(
          "success",
          `Successfully removed ${
            selectedPolicyIds?.length === 1 ? "policy" : "policies"
          }.`
        );
        setResetSelectedRows(true);
        refetchPolicies(id);
      });
    } catch {
      renderFlash(
        "error",
        `Unable to remove ${
          selectedPolicyIds?.length === 1 ? "policy" : "policies"
        }. Please try again.`
      );
    } finally {
      toggleRemovePoliciesModal();
      setIsRemovingPolicy(false);
    }
  };

  const inheritedPoliciesButtonText = (
    showPolicies: boolean,
    count: number
  ) => {
    return `${showPolicies ? "Hide" : "Show"} ${count} inherited ${
      count > 1 ? "policies" : "policy"
    }`;
  };

  const showTeamDescription = isPremiumTier && !!teamId;

  const showInheritedPoliciesButton =
    !!teamId &&
    !isFetchingTeamPolicies &&
    !teamPoliciesError &&
    !isFetchingGlobalPolicies &&
    !globalPoliciesError &&
    !!globalPolicies?.length;

  const availablePoliciesForAutomation =
    (teamId ? teamPolicies : globalPolicies) || [];

  // If team_id from URL query params is not valid, we instead use a default team
  // either the current team (if any) or all teams (for global users) or
  // the first available team (for non-global users)
  const getValidatedTeamId = () => {
    if (findAvailableTeam(teamId)) {
      return teamId;
    }
    if (!teamId && currentTeam) {
      return currentTeam.id;
    }
    if (!teamId && !currentTeam && !isOnGlobalTeam && availableTeams) {
      return availableTeams[0]?.id;
    }
    return 0;
  };

  // If team_id or currentTeam doesn't match validated id, switch to validated id
  useEffect(() => {
    if (availableTeams) {
      const validatedId = getValidatedTeamId();

      if (validatedId !== currentTeam?.id || validatedId !== teamId) {
        handleTeamSelect(validatedId);
      }
    }
  }, [availableTeams]);

  const showCtaButtons =
    (!!teamId && teamPolicies) || (!teamId && globalPolicies);

  return !availableTeams ? (
    <Spinner />
  ) : (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Policies</h1>}
                {isPremiumTier &&
                  (availableTeams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      currentUserTeams={availableTeams || []}
                      selectedTeamId={teamId}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  availableTeams.length === 1 && (
                    <h1>{availableTeams[0].name}</h1>
                  )}
              </div>
            </div>
          </div>
          {showCtaButtons && (
            <div className={`${baseClass} button-wrap`}>
              {canManageAutomations &&
                !isFetchingAutomationsConfig &&
                !isFetchingGlobalPolicies && (
                  <Button
                    onClick={toggleManageAutomationsModal}
                    className={`${baseClass}__manage-automations button`}
                    variant="inverse"
                  >
                    <span>Manage automations</span>
                  </Button>
                )}
              {canAddOrRemovePolicy && (
                <div className={`${baseClass}__action-button-container`}>
                  <Button
                    variant="brand"
                    className={`${baseClass}__select-policy-button`}
                    onClick={onAddPolicyClick}
                  >
                    Add a policy
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          {showTeamDescription ? (
            <p>
              Add additional policies for <b>all hosts assigned to this team</b>
              .
            </p>
          ) : (
            <p>
              Add policies for <b>all of your hosts</b> to see which pass your
              organization’s standards.
            </p>
          )}
        </div>
        <div>
          {!!teamId && teamPoliciesError && <TableDataError />}
          {!!teamId &&
            !teamPoliciesError &&
            (isFetchingTeamPolicies && isFetchingAutomationsConfig ? (
              <Spinner />
            ) : (
              <PoliciesListWrapper
                policiesList={teamPolicies || []}
                isLoading={
                  isFetchingTeamPolicies && isFetchingAutomationsConfig
                }
                onAddPolicyClick={onAddPolicyClick}
                onRemovePoliciesClick={onRemovePoliciesClick}
                canAddOrRemovePolicy={canAddOrRemovePolicy}
                currentTeam={currentTeam}
                currentAutomatedPolicies={currentAutomatedPolicies}
              />
            ))}
          {!teamId && globalPoliciesError && <TableDataError />}
          {!teamId &&
            !globalPoliciesError &&
            (isFetchingGlobalPolicies ? (
              <Spinner />
            ) : (
              <PoliciesListWrapper
                policiesList={globalPolicies || []}
                isLoading={
                  isFetchingGlobalPolicies && isFetchingAutomationsConfig
                }
                onAddPolicyClick={onAddPolicyClick}
                onRemovePoliciesClick={onRemovePoliciesClick}
                canAddOrRemovePolicy={canAddOrRemovePolicy}
                currentTeam={currentTeam}
                currentAutomatedPolicies={currentAutomatedPolicies}
              />
            ))}
        </div>
        {showInheritedPoliciesButton && globalPolicies && (
          <RevealButton
            isShowing={showInheritedPolicies}
            baseClass={baseClass}
            hideText={inheritedPoliciesButtonText(
              showInheritedPolicies,
              globalPolicies.length
            )}
            showText={inheritedPoliciesButtonText(
              showInheritedPolicies,
              globalPolicies.length
            )}
            caretPosition={"before"}
            tooltipHtml={
              '"All teams" policies are checked <br/> for this team’s hosts.'
            }
            onClick={toggleShowInheritedPolicies}
          />
        )}
        {showInheritedPoliciesButton && showInheritedPolicies && (
          <div className={`${baseClass}__inherited-policies-table`}>
            {globalPoliciesError && <TableDataError />}
            {!globalPoliciesError &&
              (isFetchingGlobalPolicies ? (
                <Spinner />
              ) : (
                <PoliciesListWrapper
                  isLoading={
                    isFetchingGlobalPolicies && isFetchingAutomationsConfig
                  }
                  policiesList={globalPolicies || []}
                  onRemovePoliciesClick={noop}
                  resultsTitle="policies"
                  canAddOrRemovePolicy={canAddOrRemovePolicy}
                  tableType="inheritedPolicies"
                  currentTeam={currentTeam}
                  currentAutomatedPolicies={currentAutomatedPolicies}
                />
              ))}
          </div>
        )}
        {automationsConfig && showManageAutomationsModal && (
          <ManageAutomationsModal
            automationsConfig={automationsConfig}
            availablePolicies={availablePoliciesForAutomation}
            isAutomationsLoading={isAutomationsLoading}
            showPreviewPayloadModal={showPreviewPayloadModal}
            onExit={toggleManageAutomationsModal}
            handleSubmit={handleUpdateAutomations}
            togglePreviewPayloadModal={togglePreviewPayloadModal}
          />
        )}
        {showAddPolicyModal && (
          <AddPolicyModal
            onCancel={toggleAddPolicyModal}
            router={router}
            teamId={teamId}
            teamName={currentTeam?.name}
          />
        )}
        {showRemovePoliciesModal && (
          <RemovePoliciesModal
            isLoading={isRemovingPolicy}
            onCancel={toggleRemovePoliciesModal}
            onSubmit={onRemovePoliciesSubmit}
          />
        )}
      </div>
    </div>
  );
};

export default ManagePolicyPage;
