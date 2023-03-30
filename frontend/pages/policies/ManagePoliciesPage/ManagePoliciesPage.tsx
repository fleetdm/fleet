import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import { noop } from "lodash";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IConfig, IWebhookSettings } from "interfaces/config";
import { IIntegrations } from "interfaces/integration";
import {
  IPolicyStats,
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
} from "interfaces/policy";
import { ITeamConfig } from "interfaces/team";

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
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isFreeTier,
    isPremiumTier,
    setConfig,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const {
    currentTeamId,
    currentTeamName,
    currentTeamSummary,
    isAnyTeamSelected,
    isTeamAdmin,
    isTeamMaintainer,
    isRouteOk,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: false,
    },
  });
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const { setResetSelectedRows } = useContext(TableContext);
  const [isUpdatingAutomations, setIsUpdatingAutomations] = useState(false);
  const [isUpdatingPolicies, setIsUpdatingPolicies] = useState(false);
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<number[]>([]);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showDeletePolicyModal, setShowDeletePolicyModal] = useState(false);
  const [showInheritedPolicies, setShowInheritedPolicies] = useState(false);

  const [teamPolicies, setTeamPolicies] = useState<IPolicyStats[]>();
  const [inheritedPolicies, setInheritedPolicies] = useState<IPolicyStats[]>();

  useEffect(() => {
    setLastEditedQueryPlatform(null);
  }, []);

  const {
    data: globalPolicies,
    error: globalPoliciesError,
    isFetching: isFetchingGlobalPolicies,
    refetch: refetchGlobalPolicies,
  } = useQuery<ILoadAllPoliciesResponse, Error, IPolicyStats[]>(
    ["globalPolicies", teamIdForApi],
    () => {
      return globalPoliciesAPI.loadAll();
    },
    {
      enabled: isRouteOk && !teamIdForApi,
      select: (data) => data.policies,
      staleTime: 5000,
    }
  );

  const {
    error: teamPoliciesError,
    isFetching: isFetchingTeamPolicies,
    refetch: refetchTeamPolicies,
  } = useQuery<ILoadTeamPoliciesResponse, Error, ILoadTeamPoliciesResponse>(
    ["teamPolicies", teamIdForApi],
    () => teamPoliciesAPI.loadAll(teamIdForApi),
    {
      enabled: isRouteOk && isPremiumTier && !!teamIdForApi,
      onSuccess: (data) => {
        setTeamPolicies(data.policies);
        setInheritedPolicies(data.inherited_policies);
      },
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
    ["teams", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled: isRouteOk && !!teamIdForApi && canAddOrDeletePolicy,
      select: (data) => data.team,
    }
  );

  const refetchPolicies = (teamId?: number) => {
    refetchGlobalPolicies();
    if (teamId) {
      refetchTeamPolicies();
    }
  };

  // const findAvailableTeam = (id: number) => {
  //   return availableTeams?.find((t) => t.id === id);
  // };

  const onTeamChange = useCallback(
    (teamId: number) => {
      setShowInheritedPolicies(false);
      setSelectedPolicyIds([]);
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

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

  const handleUpdateAutomations = async (requestBody: {
    webhook_settings: Pick<IWebhookSettings, "failing_policies_webhook">;
    integrations: IIntegrations;
  }) => {
    setIsUpdatingAutomations(true);
    try {
      await (isAnyTeamSelected
        ? teamsAPI.update(requestBody, teamIdForApi)
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
      isAnyTeamSelected && refetchTeamConfig();
    }
  };

  const onAddPolicyClick = () => {
    setLastEditedQueryName("");
    setLastEditedQueryDescription("");
    setLastEditedQueryResolution("");
    setLastEditedQueryCritical(false);
    toggleAddPolicyModal();
  };

  const onDeletePolicyClick = (selectedTableIds: number[]): void => {
    toggleDeletePolicyModal();
    setSelectedPolicyIds(selectedTableIds);
  };

  const onDeletePolicySubmit = async () => {
    setIsUpdatingPolicies(true);
    try {
      const request = isAnyTeamSelected
        ? teamPoliciesAPI.destroy(teamIdForApi, selectedPolicyIds)
        : globalPoliciesAPI.destroy(selectedPolicyIds);

      await request.then(() => {
        renderFlash(
          "success",
          `Successfully deleted ${
            selectedPolicyIds?.length === 1 ? "policy" : "policies"
          }.`
        );
        setResetSelectedRows(true);
        refetchPolicies(teamIdForApi);
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

  const showTeamDescription = isPremiumTier && isAnyTeamSelected;

  const showInheritedPoliciesButton =
    isAnyTeamSelected &&
    !isFetchingTeamPolicies &&
    !teamPoliciesError &&
    !isFetchingGlobalPolicies &&
    !globalPoliciesError &&
    !!globalPolicies?.length;

  const availablePoliciesForAutomation =
    (isAnyTeamSelected ? teamPolicies : globalPolicies) || [];

  const showCtaButtons =
    (isAnyTeamSelected && teamPolicies) ||
    (!isAnyTeamSelected && globalPolicies);

  const automationsConfig = isAnyTeamSelected ? teamConfig : config;

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

  return !isRouteOk || (isPremiumTier && !userTeams) ? (
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
                  ((userTeams && userTeams.length > 1) || isOnGlobalTeam) && (
                    <TeamsDropdown
                      currentUserTeams={userTeams || []}
                      selectedTeamId={currentTeamId}
                      onChange={onTeamChange}
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  userTeams &&
                  userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
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
              {canAddOrDeletePolicy &&
                ((isAnyTeamSelected && !isFetchingTeamPolicies) ||
                  !isFetchingGlobalPolicies) && (
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
          {isAnyTeamSelected && teamPoliciesError && <TableDataError />}
          {isAnyTeamSelected &&
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
                currentTeam={currentTeamSummary}
                currentAutomatedPolicies={currentAutomatedPolicies}
              />
            ))}
          {!isAnyTeamSelected && globalPoliciesError && <TableDataError />}
          {!isAnyTeamSelected &&
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
                currentTeam={currentTeamSummary}
                currentAutomatedPolicies={currentAutomatedPolicies}
              />
            ))}
        </div>
        {showInheritedPoliciesButton && globalPolicies && (
          <RevealButton
            isShowing={showInheritedPolicies}
            className={baseClass}
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
                  isLoading={isFetchingTeamPolicies}
                  policiesList={inheritedPolicies || []}
                  onDeletePolicyClick={noop}
                  canAddOrDeletePolicy={canAddOrDeletePolicy}
                  tableType="inheritedPolicies"
                  currentTeam={currentTeamSummary}
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
            teamId={teamIdForApi || 0}
            teamName={currentTeamName}
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
