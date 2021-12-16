import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useDispatch } from "react-redux";
import { find, noop } from "lodash";

// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import PATHS from "router/paths";

import { DEFAULT_POLICY } from "utilities/constants";
import { IPolicy, IPolicyStats } from "interfaces/policy";
import { ITeam } from "interfaces/team";
import { IUser } from "interfaces/user";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";

import globalPoliciesAPI from "services/entities/global_policies";
import teamsAPI from "services/entities/teams";
import teamPoliciesAPI from "services/entities/team_policies";

import { inMilliseconds, secondsToHms } from "fleet/helpers";
import sortUtils from "utilities/sort";
import permissionsUtils from "utilities/permissions";

import TableDataError from "components/TableDataError";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import IconToolTip from "components/IconToolTip";
import TeamsDropdown from "components/TeamsDropdown";
import PoliciesListWrapper from "./components/PoliciesListWrapper";
import AddPolicyModal from "./components/AddPolicyModal";
import RemovePoliciesModal from "./components/RemovePoliciesModal";

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
  router: any;
  location: any;
}): JSX.Element => {
  const { location, router } = managePoliciesPageProps;

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

  const { data: teams, isLoading: isLoadingTeams } = useQuery<
    { teams: ITeam[] },
    Error,
    ITeam[]
  >(["teams"], () => teamsAPI.loadAll({}), {
    enabled: !!isPremiumTier,
    select: (data) => data.teams,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // ===== local state
  const [globalPolicies, setGlobalPolicies] = useState<
    IPolicyStats[] | never[]
  >([]);
  const [isLoadingGlobalPolicies, setIsLoadingGlobalPolicies] = useState(true);
  const [isGlobalPoliciesError, setIsGlobalPoliciesError] = useState(false);
  const [teamPolicies, setTeamPolicies] = useState<IPolicyStats[] | never[]>(
    []
  );
  const [isLoadingTeamPolicies, setIsLoadingTeamPolicies] = useState(true);
  const [isTeamPoliciesError, setIsTeamPoliciesError] = useState(false);
  const [userTeams, setUserTeams] = useState<ITeam[] | never[] | null>(null);
  const [selectedTeamId, setSelectedTeamId] = useState<number>(
    parseInt(location?.query?.team_id, 10) || 0
  );
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<
    number[] | never[]
  >([]);
  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showRemovePoliciesModal, setShowRemovePoliciesModal] = useState(false);
  const [showInheritedPolicies, setShowInheritedPolicies] = useState(false);
  const [updateInterval, setUpdateInterval] = useState<string>(
    "osquery policy update interval"
  );
  // ===== local state

  const getGlobalPolicies = useCallback(async () => {
    setIsLoadingGlobalPolicies(true);
    setIsGlobalPoliciesError(false);
    let result;
    try {
      result = await globalPoliciesAPI
        .loadAll()
        .then((response) => response.policies);
      setGlobalPolicies(result);
      setLastEditedQueryPlatform("");
    } catch (error) {
      console.log(error);
      setIsGlobalPoliciesError(true);
    } finally {
      setIsLoadingGlobalPolicies(false);
    }
    return result;
  }, []);

  const getTeamPolicies = useCallback(async (teamId) => {
    setIsLoadingTeamPolicies(true);
    setIsTeamPoliciesError(false);
    let result;
    try {
      result = await teamPoliciesAPI
        .loadAll(teamId)
        .then((response) => response.policies);
      setTeamPolicies(result);
    } catch (error) {
      console.log(error);
      setIsTeamPoliciesError(true);
    } finally {
      setIsLoadingTeamPolicies(false);
    }
    return result;
  }, []);

  const getPolicies = useCallback(
    (teamId) => {
      return teamId ? getTeamPolicies(teamId) : getGlobalPolicies();
    },
    [getGlobalPolicies, getTeamPolicies]
  );

  const handleTeamSelect = (id: number) => {
    const { MANAGE_POLICIES } = PATHS;
    const path = id ? `${MANAGE_POLICIES}?team_id=${id}` : MANAGE_POLICIES;
    router.replace(path);
    setShowInheritedPolicies(false);
    setSelectedPolicyIds([]);
    const selectedTeam = find(teams, ["id", id]);
    setCurrentTeam(selectedTeam);
  };

  const toggleAddPolicyModal = () => setShowAddPolicyModal(!showAddPolicyModal);

  const toggleRemovePoliciesModal = () =>
    setShowRemovePoliciesModal(!showRemovePoliciesModal);

  const toggleShowInheritedPolicies = () =>
    setShowInheritedPolicies(!showInheritedPolicies);

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
    try {
      const request = selectedTeamId
        ? teamPoliciesAPI.destroy(selectedTeamId, selectedPolicyIds)
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
      getPolicies(selectedTeamId);
    }
  };

  // Sort list of teams the current user has permission to access and set as userTeams.
  useEffect(() => {
    if (isPremiumTier) {
      let unsortedTeams: ITeam[] | null = null;
      if (isOnGlobalTeam && teams) {
        unsortedTeams = teams;
      } else if (!isOnGlobalTeam && currentUser?.teams) {
        unsortedTeams = currentUser.teams;
      }
      if (unsortedTeams !== null) {
        const sortedTeams = unsortedTeams.sort((a, b) =>
          sortUtils.caseInsensitiveAsc(a.name, b.name)
        );
        setUserTeams(sortedTeams);
      }
    }
  }, [currentUser, isOnGlobalTeam, isPremiumTier, teams]);

  // Watch the location url and parse team param to set selectedTeamId.
  // Note 0 is used as the id for the "All teams" option.
  // Null case is used to represent no valid id has been selected.
  useEffect(() => {
    let teamId: number | null = parseInt(location?.query?.team_id, 10) || 0;

    // If the team id does not match one in the user teams list,
    // we use a default value and change call change handler
    // to update url params with the default value.
    // We return early to guard against potential invariant condition.
    if (userTeams && !userTeams.find((t) => t.id === teamId)) {
      if (isOnGlobalTeam) {
        // For global users, default to zero (i.e. all teams).
        if (teamId === undefined && !currentTeam) {
          handleTeamSelect(0);
          return;
        }
      } else {
        // For non-global users, default to the first team in the list.
        // If there is no default team, set teamId to null so that getPolicies
        // API request will not be triggered.
        teamId = userTeams[0]?.id || null;
        if (!currentTeam && teamId) {
          handleTeamSelect(teamId);
          return;
        }
      }
    }
    // Null case must be distinguished from 0 (which is used as the id for the "All teams" option)
    // so a falsiness check cannot be used here. Null case here allows us to skip API call
    // that would be triggered on a change to selectedTeamId.
    if (currentTeam) {
      setSelectedTeamId(currentTeam.id);
    } else {
      teamId !== null && setSelectedTeamId(teamId);
    }
  }, [isOnGlobalTeam, location, userTeams]);

  // Watch for selected team changes and call getPolicies to make new policies API request.
  useEffect(() => {
    // Null case must be distinguished from 0 (which is used as the id for the "All teams" option)
    // so a falsiness check cannot be used here. Null case here allows us to skip API call.
    if (selectedTeamId !== null) {
      if (isOnGlobalTeam || isAnyTeamMaintainerOrTeamAdmin) {
        getGlobalPolicies();
      }
      if (selectedTeamId) {
        getTeamPolicies(selectedTeamId);
      }
    }
  }, [
    getGlobalPolicies,
    getTeamPolicies,
    isAnyTeamMaintainerOrTeamAdmin,
    isOnGlobalTeam,
    selectedTeamId,
  ]);

  // Pull osquery policy update interval value from config, reformat, and set as updateInterval.
  useEffect(() => {
    if (config) {
      const { osquery_policy: interval } = config;
      interval &&
        setUpdateInterval(secondsToHms(inMilliseconds(interval) / 1000));
    }
  }, [config]);

  // If the user is free tier or if there is no selected team, we show the default description.
  // We also want to check selectTeamId for the null case so that we don't render the element prematurely.
  const showDefaultDescription =
    isFreeTier || (isPremiumTier && !selectedTeamId && selectedTeamId !== null);

  // If there aren't any policies of if there are loading errors, we don't show the update interval info banner.
  // We also want to check selectTeamId for the null case so that we don't render the element prematurely.
  const showInfoBanner =
    (selectedTeamId && !isTeamPoliciesError && !!teamPolicies?.length) ||
    (!selectedTeamId &&
      selectedTeamId !== null &&
      !isGlobalPoliciesError &&
      !!globalPolicies?.length);

  // If there aren't any policies of if there are loading errors, we don't show the inherited policies button.
  const showInheritedPoliciesButton =
    !!selectedTeamId && !!globalPolicies?.length && !isGlobalPoliciesError;

  const selectedTeamData = userTeams?.find(
    (team) => selectedTeamId === team.id
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Policies</h1>}
                {isPremiumTier &&
                  userTeams &&
                  (userTeams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      selectedTeamId={selectedTeamId}
                      currentUserTeams={userTeams || []}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  userTeams &&
                  userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
              </div>
            </div>
          </div>
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
            (isTeamPoliciesError ? (
              <TableDataError />
            ) : (
              <PoliciesListWrapper
                policiesList={teamPolicies}
                isLoading={isLoadingTeamPolicies}
                onRemovePoliciesClick={onRemovePoliciesClick}
                toggleAddPolicyModal={toggleAddPolicyModal}
                canAddOrRemovePolicy={canAddOrRemovePolicy(
                  currentUser,
                  selectedTeamId
                )}
                selectedTeamData={selectedTeamData}
              />
            ))}
          {!selectedTeamId &&
            (isGlobalPoliciesError ? (
              <TableDataError />
            ) : (
              <PoliciesListWrapper
                policiesList={globalPolicies}
                isLoading={isLoadingGlobalPolicies}
                onRemovePoliciesClick={onRemovePoliciesClick}
                toggleAddPolicyModal={toggleAddPolicyModal}
                canAddOrRemovePolicy={canAddOrRemovePolicy(
                  currentUser,
                  selectedTeamId
                )}
                selectedTeamData={selectedTeamData}
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
              isLoading={isLoadingGlobalPolicies}
              policiesList={globalPolicies}
              onRemovePoliciesClick={noop}
              resultsTitle="policies"
              canAddOrRemovePolicy={canAddOrRemovePolicy(
                currentUser,
                selectedTeamId
              )}
              tableType="inheritedPolicies"
              selectedTeamData={selectedTeamData}
            />
          </div>
        )}
        {showAddPolicyModal && (
          <AddPolicyModal
            onCancel={toggleAddPolicyModal}
            router={router}
            teamId={selectedTeamId}
            teamName={selectedTeamData?.name}
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
