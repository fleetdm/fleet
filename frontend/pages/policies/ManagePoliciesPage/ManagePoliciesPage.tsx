import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import { noop } from "lodash";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";

import { IAutomationsConfig, IConfig } from "interfaces/config";
import { IPolicyStats, ILoadAllPoliciesResponse } from "interfaces/policy";
import { ITeamAutomationsConfig, ITeamConfig } from "interfaces/team";

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
import MainContent from "components/MainContent";

import PoliciesTable from "./components/PoliciesTable";
import ManageAutomationsModal from "./components/ManageAutomationsModal";
import AddPolicyModal from "./components/AddPolicyModal";
import DeletePolicyModal from "./components/DeletePolicyModal";

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
  const [isUpdatingAutomations, setIsUpdatingAutomations] = useState<boolean>(
    false
  );
  const [isUpdatingPolicies, setIsUpdatingPolicies] = useState<boolean>(false);
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<number[]>([]);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showDeletePolicyModal, setShowDeletePolicyModal] = useState(false);
  const [showInheritedPolicies, setShowInheritedPolicies] = useState(false);

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
      staleTime: 5000,
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
      staleTime: 5000,
    }
  );

  const canAddOrDeletePolicy =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer || isTeamAdmin;
  const canManageAutomations = isGlobalAdmin || isTeamAdmin;

  const {
    data: config,
    isFetching: isFetchingConfig,
    refetch: refetchConfig,
  } = useQuery<IConfig, Error>(
    ["config"],
    () => {
      return configAPI.loadAll();
    },
    {
      enabled: canAddOrDeletePolicy,
      onSuccess: (data) => {
        setConfig(data);
      },
      staleTime: 5000,
    }
  );

  const {
    data: teamConfig,
    isFetching: isFetchingTeamConfig,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["teams", teamId],
    () => teamsAPI.load(teamId),
    {
      enabled: !!teamId && canAddOrDeletePolicy,
      select: (data) => data.team,
      staleTime: 5000,
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

  const toggleDeletePolicyModal = () =>
    setShowDeletePolicyModal(!showDeletePolicyModal);

  const toggleShowInheritedPolicies = () =>
    setShowInheritedPolicies(!showInheritedPolicies);

  const handleUpdateAutomations = async (
    requestBody: IAutomationsConfig | ITeamAutomationsConfig
  ) => {
    setIsUpdatingAutomations(true);
    try {
      await (teamId
        ? teamsAPI.update(requestBody, teamId)
        : configAPI.update(requestBody));
      renderFlash("success", "Successfully updated policy automations.");
    } catch {
      renderFlash(
        "error",
        "Could not update policy automations. Please try again."
      );
    } finally {
      toggleManageAutomationsModal();
      setIsUpdatingAutomations(false);
      refetchConfig();
      teamId && refetchTeamConfig();
    }
  };

  const onAddPolicyClick = () => {
    setLastEditedQueryName("");
    setLastEditedQueryDescription("");
    setLastEditedQueryResolution("");
    toggleAddPolicyModal();
  };

  const onDeletePolicyClick = (selectedTableIds: number[]): void => {
    toggleDeletePolicyModal();
    setSelectedPolicyIds(selectedTableIds);
  };

  const onDeletePolicySubmit = async () => {
    const id = currentTeam?.id;
    setIsUpdatingPolicies(true);
    try {
      const request = id
        ? teamPoliciesAPI.destroy(id, selectedPolicyIds)
        : globalPoliciesAPI.destroy(selectedPolicyIds);

      await request.then(() => {
        renderFlash(
          "success",
          `Successfully deleted ${
            selectedPolicyIds?.length === 1 ? "policy" : "policies"
          }.`
        );
        setResetSelectedRows(true);
        refetchPolicies(id);
      });
    } catch {
      renderFlash(
        "error",
        `Unable to delete ${
          selectedPolicyIds?.length === 1 ? "policy" : "policies"
        }. Please try again.`
      );
    } finally {
      toggleDeletePolicyModal();
      setIsUpdatingPolicies(false);
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

  const automationsConfig = teamId ? teamConfig : config;

  // NOTE: backend uses webhook_settings to store automated policy ids for both webhooks and integrations
  let currentAutomatedPolicies: number[] = [];
  if (automationsConfig) {
    const {
      webhook_settings: { failing_policies_webhook: webhook },
      integrations,
    } = automationsConfig;

    let isIntegrationEnabled = false;
    if (integrations) {
      const { jira, zendesk } = integrations;
      isIntegrationEnabled =
        !!jira?.find((j) => j.enable_failing_policies) ||
        !!zendesk?.find((z) => z.enable_failing_policies);
    }

    if (isIntegrationEnabled || webhook?.enable_failing_policies_webhook) {
      currentAutomatedPolicies = webhook?.policy_ids || [];
    }
  }

  return !availableTeams ? (
    <Spinner />
  ) : (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
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
                automationsConfig &&
                !isFetchingGlobalPolicies && (
                  <Button
                    onClick={toggleManageAutomationsModal}
                    className={`${baseClass}__manage-automations button`}
                    variant="inverse"
                  >
                    <span>Manage automations</span>
                  </Button>
                )}
              {canAddOrDeletePolicy && (
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
            (isFetchingTeamPolicies ? (
              <Spinner />
            ) : (
              <PoliciesTable
                policiesList={teamPolicies || []}
                isLoading={
                  isFetchingTeamPolicies ||
                  isFetchingTeamConfig ||
                  isFetchingConfig
                }
                onAddPolicyClick={onAddPolicyClick}
                onDeletePolicyClick={onDeletePolicyClick}
                canAddOrDeletePolicy={canAddOrDeletePolicy}
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
              <PoliciesTable
                policiesList={globalPolicies || []}
                isLoading={isFetchingGlobalPolicies || isFetchingConfig}
                onAddPolicyClick={onAddPolicyClick}
                onDeletePolicyClick={onDeletePolicyClick}
                canAddOrDeletePolicy={canAddOrDeletePolicy}
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
                <PoliciesTable
                  isLoading={isFetchingGlobalPolicies}
                  policiesList={globalPolicies || []}
                  onDeletePoliciesClick={noop}
                  canAddOrDeletePolicy={canAddOrDeletePolicy}
                  tableType="inheritedPolicies"
                  currentTeam={currentTeam}
                />
              ))}
          </div>
        )}
        {config && automationsConfig && showManageAutomationsModal && (
          <ManageAutomationsModal
            automationsConfig={automationsConfig}
            availableIntegrations={config.integrations}
            availablePolicies={availablePoliciesForAutomation}
            isUpdatingAutomations={isUpdatingAutomations}
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
        {showDeletePolicyModal && (
          <DeletePolicyModal
            isUpdatingPolicies={isUpdatingPolicies}
            onCancel={toggleDeletePolicyModal}
            onSubmit={onDeletePolicySubmit}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManagePolicyPage;
