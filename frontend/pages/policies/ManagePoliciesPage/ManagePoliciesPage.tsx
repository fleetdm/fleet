import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useDispatch } from "react-redux";
import { find, noop } from "lodash";

// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import PATHS from "router/paths";

import { DEFAULT_POLICY } from "utilities/constants";
import { IPolicy, IPolicyStats } from "interfaces/policy";
import { IWebhookFailingPolicies } from "interfaces/webhook";
import { ITeam } from "interfaces/team";
import { IUser } from "interfaces/user";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";

import configAPI from "services/entities/config";
import globalPoliciesAPI, {
  IGlobalPoliciesStatsResponse,
} from "services/entities/global_policies";
import teamsAPI from "services/entities/teams";
import teamPoliciesAPI from "services/entities/team_policies";

import { inMilliseconds, secondsToHms } from "fleet/helpers";
import sortUtils from "utilities/sort";
import permissionsUtils from "utilities/permissions";

import Spinner from "components/Spinner";
import TableDataError from "components/TableDataError";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import IconToolTip from "components/IconToolTip";
import TeamsDropdown from "components/TeamsDropdown";
import PoliciesListWrapper from "./components/PoliciesListWrapper";
import ManageAutomationsModal from "./components/ManageAutomationsModal";
import AddPolicyModal from "./components/AddPolicyModal";
import RemovePoliciesModal from "./components/RemovePoliciesModal";
import { useDebouncedCallback } from "use-debounce/lib";
import { current } from "@reduxjs/toolkit";

const baseClass = "manage-policies-page";

const DOCS_LINK =
  "https://fleetdm.com/docs/deploying/configuration#osquery-policy-update-interval";

const renderInheritedPoliciesButtonText = (
  showPolicies: boolean,
  policies: IPolicy[]
) => {
  const count = policies.length;

  return `${showPolicies ? "Hide" : "Show"} ${count} inherited ${
    count > 1 ? "policies" : "policy"
  }`;
};

const ManagePolicyPage = (managePoliciesPageProps: {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  router: any;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location: any;
  params: { team_id: string };
}): JSX.Element => {
  const { location, params, router } = managePoliciesPageProps;
  const { team_id } = params;

  const dispatch = useDispatch();

  const {
    config,
    currentUser,
    isAnyTeamMaintainerOrTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isFreeTier,
    isPremiumTier,
    currentTeam,
    setCurrentTeam,
  } = useContext(AppContext);

  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const { isTeamMaintainer, isTeamAdmin } = permissionsUtils;
  const canAddOrRemovePolicy = (user: IUser | null, teamId: number | null) =>
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isTeamMaintainer(user, teamId) ||
    isTeamAdmin(user, teamId);

  const filterAndSortTeamOptions = (allTeams: ITeam[], userTeams: ITeam[]) => {
    const filteredSortedTeams = allTeams
      .sort((teamA: ITeam, teamB: ITeam) =>
        sortUtils.caseInsensitiveAsc(teamA.name, teamB.name)
      )
      .filter((team: ITeam) => {
        const userTeam = userTeams.find(
          (thisUserTeam) => thisUserTeam.id === team.id
        );
        return userTeam?.role !== "observer" ? team : null;
      });

    return filteredSortedTeams;
  };

  const [userTeams, setUserTeams] = useState<ITeam[] | never[] | null>(null);
  const { data: teams, isLoading: isLoadingTeams } = useQuery<
    { teams: ITeam[] },
    Error,
    ITeam[]
  >(["teams"], () => teamsAPI.loadAll({}), {
    enabled: !!isPremiumTier,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    select: (data) => {
      return currentUser?.teams
        ? filterAndSortTeamOptions(data.teams, currentUser.teams)
        : data.teams;
    },
    // onSuccess: (allTeams) => {
    //   setUserTeams(
    //     currentUser?.teams
    //       ? filterAndSortTeamOptions(allTeams, currentUser.teams)
    //       : allTeams
    //   );
    // },
  });

  // ===== local state
  // const [globalPolicies, setGlobalPolicies] = useState<
  //   IPolicyStats[] | never[]
  // >([]);
  // const [isLoadingGlobalPolicies, setIsLoadingGlobalPolicies] = useState(true);
  // const [isGlobalPoliciesError, setIsGlobalPoliciesError] = useState(false);
  // const [teamPolicies, setTeamPolicies] = useState<IPolicyStats[] | never[]>(
  //   []
  // );
  // const [isLoadingTeamPolicies, setIsLoadingTeamPolicies] = useState(true);
  // const [isTeamPoliciesError, setIsTeamPoliciesError] = useState(false);

  // const [selectedTeamId, setSelectedTeamId] = useState<number>(
  //   parseInt(location?.query?.team_id, 10) || 0
  // );
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<
    number[] | never[]
  >([]);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showRemovePoliciesModal, setShowRemovePoliciesModal] = useState(false);
  const [showInheritedPolicies, setShowInheritedPolicies] = useState(false);
  const [updateInterval, setUpdateInterval] = useState<string>(
    "osquery policy update interval"
  );
  const [
    isLoadingFailingPoliciesWebhook,
    setIsLoadingFailingPoliciesWebhook,
  ] = useState(true);
  const [
    isFailingPoliciesWebhookError,
    setIsFailingPoliciesWebhookError,
  ] = useState(false);
  const [failingPoliciesWebhook, setFailingPoliciesWebhook] = useState<
    IWebhookFailingPolicies | undefined
  >();
  const [currentAutomatedPolicies, setCurrentAutomatedPolicies] = useState<
    number[]
  >();
  // ===== local state

  const {
    data: globalPolicies,
    error: globalPoliciesError,
    isLoading: isLoadingGlobalPolicies,
    refetch: refetchGlobalPolicies,
  } = useQuery<IGlobalPoliciesStatsResponse, Error, IPolicyStats[]>(
    ["globalPolicies", currentTeam?.id],
    () => globalPoliciesAPI.loadAll(),
    {
      // enabled: isOnGlobalTeam,
      select: (data) => data.policies,
      onSuccess: () => setLastEditedQueryPlatform(""),
    }
  );

  const {
    data: teamPolicies,
    error: teamPoliciesError,
    isLoading: isLoadingTeamPolicies,
    refetch: refetchTeamPolicies,
  } = useQuery(
    ["teamPolicies", currentTeam?.id],
    () => !!currentTeam?.id && teamPoliciesAPI.loadAll(currentTeam.id),
    {
      enabled: isPremiumTier && !!currentTeam?.id,
      select: (data) => data.policies,
      // onSuccess: () => setLastEditedQueryPlatform(""),
    }
  );

  const refetchPolicies = (id?: number) => {
    if (id) {
      refetchTeamPolicies();
      refetchGlobalPolicies();
    } else {
      refetchGlobalPolicies();
    }
  };

  const handleTeamSelect = (teamId: number) => {
    console.log("selecting: ", teamId);
    const { MANAGE_POLICIES } = PATHS;
    const path = teamId
      ? `${MANAGE_POLICIES}?team_id=${teamId}`
      : MANAGE_POLICIES;
    router.replace(path);
    setShowInheritedPolicies(false);
    setSelectedPolicyIds([]);
    const selectedTeam = find(teams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  const getFailingPoliciesWebhook = useCallback(async () => {
    setIsLoadingFailingPoliciesWebhook(true);
    setIsFailingPoliciesWebhookError(false);
    let result;
    try {
      result = await configAPI
        .loadAll()
        .then((response) => response.webhook_settings.failing_policies_webhook);
      setFailingPoliciesWebhook(result);
      setCurrentAutomatedPolicies(result.policy_ids);
    } catch (error) {
      console.log(error);
      setIsFailingPoliciesWebhookError(true);
    } finally {
      setIsLoadingFailingPoliciesWebhook(false);
    }
    return result;
  }, []);

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

  const onManageAutomationsClick = () => {
    toggleManageAutomationsModal();
  };

  const onCreateWebhookSubmit = async ({
    destination_url,
    policy_ids,
    enable_failing_policies_webhook,
  }: IWebhookFailingPolicies) => {
    try {
      const request = configAPI.update({
        webhook_settings: {
          failing_policies_webhook: {
            destination_url,
            policy_ids,
            enable_failing_policies_webhook,
          },
        },
      });
      await request.then(() => {
        dispatch(
          renderFlash("success", "Successfully updated policy automations.")
        );
      });
    } catch {
      dispatch(
        renderFlash(
          "error",
          "Could not update policy automations. Please try again."
        )
      );
    } finally {
      toggleManageAutomationsModal();
      getFailingPoliciesWebhook();
    }
  };

  const onAddPolicyClick = () => {
    setLastEditedQueryName("");
    setLastEditedQueryDescription("");
    setLastEditedQueryBody(DEFAULT_POLICY.query);
    setLastEditedQueryResolution("");
    toggleAddPolicyModal();
  };

  const onRemovePoliciesClick = (selectedTableIds: number[]): void => {
    toggleRemovePoliciesModal();
    setSelectedPolicyIds(selectedTableIds);
  };

  const onRemovePoliciesSubmit = async () => {
    const teamId = currentTeam?.id;
    try {
      const request = teamId
        ? teamPoliciesAPI.destroy(teamId, selectedPolicyIds)
        : globalPoliciesAPI.destroy(selectedPolicyIds);

      await request.then(() => {
        dispatch(
          renderFlash(
            "success",
            `Successfully removed ${
              selectedPolicyIds?.length === 1 ? "policy" : "policies"
            }.`
          )
        );
      });
    } catch {
      dispatch(
        renderFlash(
          "error",
          `Unable to remove ${
            selectedPolicyIds?.length === 1 ? "policy" : "policies"
          }. Please try again.`
        )
      );
    } finally {
      toggleRemovePoliciesModal();
      refetchPolicies(teamId);
    }
  };

  // Sort list of teams the current user has permission to access and set as userTeams.
  // useEffect(() => {
  //   if (isPremiumTier) {
  //     let unsortedTeams: ITeam[] | null = null;
  //     if (isOnGlobalTeam && teams) {
  //       unsortedTeams = teams;
  //     } else if (!isOnGlobalTeam && currentUser?.teams) {
  //       unsortedTeams = currentUser.teams;
  //     }
  //     if (unsortedTeams !== null) {
  //       const sortedTeams = unsortedTeams.sort((a, b) =>
  //         sortUtils.caseInsensitiveAsc(a.name, b.name)
  //       );
  //       setTeamOptions(sortedTeams);
  //     }
  //   }
  // }, [currentUser, isOnGlobalTeam, isPremiumTier, teams]);

  // Watch the location url and parse team param to set selectedTeamId.
  // Note 0 is used as the id for the "All teams" option.
  // Null case is used to represent no valid id has been selected.
  // useEffect(() => {
  //   let teamId: number | null = parseInt(location?.query?.team_id, 10) || 0;

  //   // If the team id does not match one in the user teams list,
  //   // we use a default value and change call change handler
  //   // to update url params with the default value.
  //   // We return early to guard against potential invariant condition.
  //   if (userTeams && !userTeams.find((t) => t.id === teamId)) {
  //     if (isOnGlobalTeam) {
  //       // For global users, default to zero (i.e. all teams).
  //       if (teamId === undefined && !currentTeam) {
  //         handleTeamSelect(0);
  //         return;
  //       }
  //     } else {
  //       // For non-global users, default to the first team in the list.
  //       // If there is no default team, set teamId to null so that getPolicies
  //       // API request will not be triggered.
  //       teamId = userTeams[0]?.id || null;
  //       if (!currentTeam && teamId) {
  //         handleTeamSelect(teamId);
  //         return;
  //       }
  //     }
  //   }
  //   // Null case must be distinguished from 0 (which is used as the id for the "All teams" option)
  //   // so a falsiness check cannot be used here. Null case here allows us to skip API call
  //   // that would be triggered on a change to selectedTeamId.
  //   if (currentTeam) {
  //     setSelectedTeamId(currentTeam.id);
  //   } else {
  //     teamId !== null && setSelectedTeamId(teamId);
  //   }
  // }, [isOnGlobalTeam, location, userTeams]);

  // // Watch for selected team changes and call getPolicies to make new policies API request.
  // useEffect(() => {
  //   // Null case must be distinguished from 0 (which is used as the id for the "All teams" option)
  //   // so a falsiness check cannot be used here. Null case here allows us to skip API call.
  //   if (selectedTeamId !== null) {
  //     if (isOnGlobalTeam || isAnyTeamMaintainerOrTeamAdmin) {
  //       getGlobalPolicies();
  //     }
  //     if (selectedTeamId) {
  //       getTeamPolicies(selectedTeamId);
  //     }
  //   }
  //   getFailingPoliciesWebhook();
  // }, [
  //   getGlobalPolicies,
  //   getTeamPolicies,
  //   getFailingPoliciesWebhook,
  //   isAnyTeamMaintainerOrTeamAdmin,
  //   isOnGlobalTeam,
  //   selectedTeamId,
  // ]);

  // Pull osquery policy update interval value from config, reformat, and set as updateInterval.
  useEffect(() => {
    if (config) {
      const { osquery_policy: interval } = config;
      interval &&
        setUpdateInterval(secondsToHms(inMilliseconds(interval) / 1000));
    }
  }, [config]);

  let selectedTeamId: number;

  if (currentTeam) {
    selectedTeamId = currentTeam.id;
  } else {
    selectedTeamId = team_id ? parseInt(team_id, 10) : 0;
  }

  if (!isOnGlobalTeam && !selectedTeamId && teams) {
    handleTeamSelect(teams[0].id);
  }

  // If the user is free tier or if there is no selected team, we show the default description.
  // We also want to check selectTeamId for the null case so that we don't render the element prematurely.
  const showDefaultDescription =
    isFreeTier ||
    (isPremiumTier && !selectedTeamId && selectedTeamId !== undefined);

  const showInfoBanner =
    (selectedTeamId && !teamPoliciesError && !!teamPolicies?.length) ||
    (!selectedTeamId &&
      selectedTeamId !== null &&
      !globalPoliciesError &&
      !!globalPolicies?.length);

  const showInheritedPoliciesButton =
    !!selectedTeamId && !!globalPolicies?.length && !globalPoliciesError;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Policies</h1>}
                {isPremiumTier &&
                  teams &&
                  (teams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      currentUserTeams={teams || []}
                      selectedTeamId={selectedTeamId}
                      // includeAll={isOnGlobalTeam}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  teams &&
                  teams.length === 1 && <h1>{teams[0].name}</h1>}
              </div>
            </div>
          </div>
          <div className={`${baseClass} button-wrap`}>
            {canAddOrRemovePolicy(currentUser, selectedTeamId) &&
              selectedTeamId === 0 && (
                <Button
                  onClick={() => onManageAutomationsClick()}
                  className={`${baseClass}__manage-automations button`}
                  variant="inverse"
                >
                  <span>Manage automations</span>
                </Button>
              )}
            {canAddOrRemovePolicy(currentUser, selectedTeamId) && (
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
        </div>
        {!isLoadingTeams && (
          <div className={`${baseClass}__description`}>
            {isPremiumTier && !!selectedTeamId && (
              <p>
                Add additional policies for{" "}
                <b>all hosts assigned to this team</b>.
              </p>
            )}
            {showDefaultDescription && (
              <p>
                Add policies for <b>all of your hosts</b> to see which pass your
                organization’s standards.{" "}
              </p>
            )}
          </div>
        )}
        {!!updateInterval && showInfoBanner && (
          <InfoBanner className={`${baseClass}__sandbox-info`}>
            <p>
              Your policies are checked every <b>{updateInterval.trim()}</b>.{" "}
              {isGlobalAdmin && (
                <span>
                  Check out the Fleet documentation on{" "}
                  <a href={DOCS_LINK} target="_blank" rel="noreferrer">
                    <b>how to edit this frequency</b>
                  </a>
                  .
                </span>
              )}
            </p>
          </InfoBanner>
        )}
        <div>
          {!!selectedTeamId &&
            (teamPoliciesError ? (
              <TableDataError />
            ) : (
              <PoliciesListWrapper
                policiesList={teamPolicies}
                isLoading={
                  isLoadingTeamPolicies && isLoadingFailingPoliciesWebhook
                }
                onRemovePoliciesClick={onRemovePoliciesClick}
                canAddOrRemovePolicy={canAddOrRemovePolicy(
                  currentUser,
                  selectedTeamId
                )}
                currentTeam={currentTeam}
                currentAutomatedPolicies={currentAutomatedPolicies}
              />
            ))}
          {!selectedTeamId &&
            (globalPoliciesError ? (
              <TableDataError />
            ) : (
              <PoliciesListWrapper
                policiesList={globalPolicies || []}
                isLoading={
                  isLoadingGlobalPolicies && isLoadingFailingPoliciesWebhook
                }
                onRemovePoliciesClick={onRemovePoliciesClick}
                canAddOrRemovePolicy={canAddOrRemovePolicy(
                  currentUser,
                  selectedTeamId
                )}
                currentTeam={currentTeam}
                currentAutomatedPolicies={currentAutomatedPolicies}
              />
            ))}
        </div>
        {showInheritedPoliciesButton && (
          <span>
            <Button
              variant="unstyled"
              className={`${showInheritedPolicies ? "upcarat" : "rightcarat"} 
                     ${baseClass}__inherited-policies-button`}
              onClick={toggleShowInheritedPolicies}
            >
              {renderInheritedPoliciesButtonText(
                showInheritedPolicies,
                globalPolicies
              )}
            </Button>
            <div className={`${baseClass}__details`}>
              <IconToolTip
                isHtml
                text={
                  "\
              <center><p>“All teams” policies are checked <br/> for this team’s hosts.</p></center>\
            "
                }
              />
            </div>
          </span>
        )}
        {showInheritedPoliciesButton && showInheritedPolicies && (
          <div className={`${baseClass}__inherited-policies-table`}>
            <PoliciesListWrapper
              isLoading={
                isLoadingGlobalPolicies && isLoadingFailingPoliciesWebhook
              }
              policiesList={globalPolicies || []}
              onRemovePoliciesClick={noop}
              resultsTitle="policies"
              canAddOrRemovePolicy={canAddOrRemovePolicy(
                currentUser,
                selectedTeamId
              )}
              tableType="inheritedPolicies"
              currentTeam={currentTeam}
              currentAutomatedPolicies={currentAutomatedPolicies}
            />
          </div>
        )}
        {showManageAutomationsModal && (
          <ManageAutomationsModal
            onCancel={toggleManageAutomationsModal}
            onCreateWebhookSubmit={onCreateWebhookSubmit}
            togglePreviewPayloadModal={togglePreviewPayloadModal}
            showPreviewPayloadModal={showPreviewPayloadModal}
            availablePolicies={globalPolicies || []}
            currentAutomatedPolicies={currentAutomatedPolicies || []}
            currentDestinationUrl={
              (failingPoliciesWebhook &&
                failingPoliciesWebhook.destination_url) ||
              ""
            }
          />
        )}
        {showAddPolicyModal && (
          <AddPolicyModal
            onCancel={toggleAddPolicyModal}
            router={router}
            teamId={selectedTeamId}
            teamName={currentTeam?.name}
          />
        )}
        {showRemovePoliciesModal && (
          <RemovePoliciesModal
            onCancel={toggleRemovePoliciesModal}
            onSubmit={onRemovePoliciesSubmit}
          />
        )}
      </div>
    </div>
  );
};

export default ManagePolicyPage;
