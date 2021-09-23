import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useDispatch } from "react-redux";
import { noop } from "lodash";

// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import PATHS from "router/paths";

import { IPolicy } from "interfaces/policy";
import { ITeam } from "interfaces/team";

import { AppContext } from "context/app";

import fleetQueriesAPI from "services/entities/queries";
import globalPoliciesAPI from "services/entities/global_policies";
import teamsAPI from "services/entities/teams";
import teamPoliciesAPI from "services/entities/team_policies";

import { inMilliseconds, secondsToHms } from "fleet/helpers";
import sortUtils from "utilities/sort";

import TableDataError from "components/TableDataError";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import PoliciesListWrapper from "./components/PoliciesListWrapper";
import AddPolicyModal from "./components/AddPolicyModal";
import RemovePoliciesModal from "./components/RemovePoliciesModal";
import TeamsDropdown from "./components/TeamsDropdown";

const baseClass = "manage-policies-page";

const DOCS_LINK =
  "https://fleetdm.com/docs/deploying/configuration#osquery_detail_update_interval";

const INHERITED_POLICIES_COUNT_HTML = (
  <span>
    {" "}
    inherited from{" "}
    <span className={`${baseClass}__vibrant-blue`}>All teams policy</span>
  </span>
);

const ManagePolicyPage = (managePoliciesPageProps: {
  router: any;
  location: any;
}): JSX.Element => {
  const { location, router } = managePoliciesPageProps;
  const dispatch = useDispatch();

  const {
    config,
    currentUser,
    isAnyTeamMaintainer,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isOnlyObserver,
    isFreeTier,
    isPremiumTier,
  } = useContext(AppContext);

  const {
    isLoading: isTeamsLoading,
    data: teams,
    error: teamsError,
  } = useQuery(["teams"], () => teamsAPI.loadAll({}), {
    enabled: !!isPremiumTier,
    select: (data) => data.teams,
  });

  const {
    isLoading: isFleetQueriesLoading,
    data: fleetQueries,
    error: fleetQueriesError,
  } = useQuery(["fleetQueries"], () => fleetQueriesAPI.loadAll(), {
    select: (data) => data.queries,
  });

  const [globalPolicies, setGlobalPolicies] = useState<IPolicy[] | never[]>([]);
  const [isLoadingGlobalPolicies, setIsLoadingGlobalPolicies] = useState(true);
  const [isGlobalPoliciesError, setIsGlobalPoliciesError] = useState(false);

  const [teamPolicies, setTeamPolicies] = useState<IPolicy[] | never[]>([]);
  const [isLoadingTeamPolicies, setIsLoadingTeamPolicies] = useState(true);
  const [isTeamPoliciesError, setIsTeamPoliciesError] = useState(false);

  const [userTeams, setUserTeams] = useState<ITeam[] | never[] | null>(null);
  const [selectedTeamId, setSelectedTeamId] = useState<number | null>(
    parseInt(location?.query?.team_id, 10) || null
  );
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<
    number[] | never[]
  >([]);

  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showRemovePoliciesModal, setShowRemovePoliciesModal] = useState(false);
  const [showInheritedPolicies, setShowInheritedPolicies] = useState(false);

  const [updateInterval, setUpdateInterval] = useState<string>(
    "osquery detail update interval"
  );

  const getGlobalPolicies = useCallback(async () => {
    setIsLoadingGlobalPolicies(true);
    setIsGlobalPoliciesError(false);
    let result;
    try {
      result = await globalPoliciesAPI
        .loadAll()
        .then((response) => response.policies);
      setGlobalPolicies(result);
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

  const handleChangeSelectedTeam = useCallback(
    (id: number) => {
      const { MANAGE_POLICIES } = PATHS;
      const path = id ? `${MANAGE_POLICIES}?team_id=${id}` : MANAGE_POLICIES;
      router.replace(path);
      setShowInheritedPolicies(false);
      setSelectedPolicyIds([]);
    },
    [router]
  );

  const toggleAddPolicyModal = useCallback(() => {
    setShowAddPolicyModal(!showAddPolicyModal);
  }, [showAddPolicyModal, setShowAddPolicyModal]);

  const toggleRemovePoliciesModal = useCallback(() => {
    setShowRemovePoliciesModal(!showRemovePoliciesModal);
  }, [showRemovePoliciesModal, setShowRemovePoliciesModal]);

  const toggleShowInheritedPolicies = useCallback(() => {
    setShowInheritedPolicies(!showInheritedPolicies);
  }, [showInheritedPolicies, setShowInheritedPolicies]);

  const onRemovePoliciesClick = useCallback(
    (selectedTableIds: number[]): void => {
      toggleRemovePoliciesModal();
      setSelectedPolicyIds(selectedTableIds);
    },
    [toggleRemovePoliciesModal]
  );

  const onRemovePoliciesSubmit = useCallback(() => {
    const request = selectedTeamId
      ? teamPoliciesAPI.destroy(selectedTeamId, selectedPolicyIds)
      : globalPoliciesAPI.destroy(selectedPolicyIds);

    request
      .then(() => {
        dispatch(
          renderFlash(
            "success",
            `Successfully removed ${
              selectedPolicyIds && selectedPolicyIds.length === 1
                ? "policy"
                : "policies"
            }.`
          )
        );
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Unable to remove ${
              selectedPolicyIds && selectedPolicyIds.length === 1
                ? "policy"
                : "policies"
            }. Please try again.`
          )
        );
      })
      .finally(() => {
        toggleRemovePoliciesModal();
        getPolicies(selectedTeamId);
      });
  }, [
    dispatch,
    getPolicies,
    selectedPolicyIds,
    selectedTeamId,
    toggleRemovePoliciesModal,
  ]);

  const onAddPolicySubmit = useCallback(
    (query_id: number | undefined) => {
      if (!query_id) {
        dispatch(
          renderFlash("error", "Could not add policy. Please try again.")
        );
        console.log("Missing query id; cannot add policy");
        return false;
      }
      const request = selectedTeamId
        ? teamPoliciesAPI.create(selectedTeamId, query_id)
        : globalPoliciesAPI.create(query_id);

      request
        .then(() => {
          dispatch(renderFlash("success", `Successfully added policy.`));
        })
        .catch(() => {
          dispatch(
            renderFlash("error", "Could not add policy. Please try again.")
          );
        })
        .finally(() => {
          toggleAddPolicyModal();
          getPolicies(selectedTeamId);
        });
      return false;
    },
    [dispatch, getPolicies, selectedTeamId, toggleAddPolicyModal]
  );

  // Sort list of teams the current user has permission to access and set as userTeams.
  useEffect(() => {
    if (isPremiumTier) {
      let unsortedTeams: ITeam[] | null = null;
      if (isOnGlobalTeam && teams) {
        unsortedTeams = teams;
      } else if (!isOnGlobalTeam && currentUser && currentUser.teams) {
        unsortedTeams = currentUser.teams;
      }
      if (unsortedTeams !== null) {
        setUserTeams(
          unsortedTeams.sort((a, b) =>
            sortUtils.caseInsensitiveAsc(b.name, a.name)
          )
        );
      }
    }
  }, [currentUser, isOnGlobalTeam, isPremiumTier, teams]);

  // Parse url query param and set as selectedTeamId.
  useEffect(() => {
    let teamId: number | null = parseInt(location?.query?.team_id, 10) || 0;

    // If the team id does not match one in the user teams list, set a default value.
    if (userTeams && !userTeams.find((t) => t.id === teamId)) {
      if (isOnGlobalTeam) {
        // For global users, default to zero (i.e. all teams).
        teamId !== 0 && handleChangeSelectedTeam(0);
        teamId = 0;
      } else {
        // For non-global users, default to the first team in the list.
        // If there is no default team, set teamId to null so that getPolicies
        // API request will not be triggered.
        teamId = userTeams[0]?.id || null;
        teamId && handleChangeSelectedTeam(teamId);
      }
    }
    teamId !== null && setSelectedTeamId(teamId);
  }, [handleChangeSelectedTeam, isOnGlobalTeam, location, userTeams]);

  // Watch for selected team changes and call getPolicies to make new policies API request.
  useEffect(() => {
    if (selectedTeamId !== null) {
      getGlobalPolicies();
      if (selectedTeamId) {
        getTeamPolicies(selectedTeamId);
      }
    }
  }, [getGlobalPolicies, getTeamPolicies, selectedTeamId]);

  // Pull osquery detail update interval value from config, reformat, and set as updateInterval.
  useEffect(() => {
    if (config) {
      const { osquery_detail: interval } = config;
      interval &&
        setUpdateInterval(secondsToHms(inMilliseconds(interval) / 1000));
    }
  }, [config]);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Policies</h1>}
                {isPremiumTier &&
                  userTeams !== null &&
                  selectedTeamId !== null && (
                    <TeamsDropdown
                      currentUserTeams={userTeams}
                      onChange={handleChangeSelectedTeam}
                      selectedTeam={selectedTeamId}
                    />
                  )}
              </div>
            </div>
          </div>
          {(isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainer) && (
            <div className={`${baseClass}__action-button-container`}>
              <Button
                variant="brand"
                className={`${baseClass}__add-policy-button`}
                onClick={toggleAddPolicyModal}
              >
                Add a policy
              </Button>
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          {isPremiumTier && !!selectedTeamId && (
            <p>
              Add additional policies for <b>all hosts assigned to this team</b>
              .
            </p>
          )}
          {isFreeTier ||
            (isPremiumTier && !selectedTeamId && selectedTeamId !== null && (
              <p>
                Add policies for <b>all of your hosts</b> to see which pass your
                organization’s standards.{" "}
              </p>
            ))}
        </div>
        {updateInterval &&
          ((selectedTeamId && !isTeamPoliciesError && !!teamPolicies?.length) ||
            (!selectedTeamId &&
              selectedTeamId !== null &&
              !isGlobalPoliciesError &&
              !!globalPolicies?.length)) && (
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
                selectedTeamId={selectedTeamId}
                showSelectionColumn={!isOnlyObserver}
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
                selectedTeamId={selectedTeamId}
                showSelectionColumn={!isOnlyObserver}
              />
            ))}
        </div>
        {!!selectedTeamId &&
          !!teamPolicies?.length &&
          !!globalPolicies?.length && (
            <span>
              <Button
                variant="unstyled"
                className={`${showInheritedPolicies ? "upcarat" : "rightcarat"} 
                     ${baseClass}__inherited-policies-button`}
                onClick={toggleShowInheritedPolicies}
              >
                {`${
                  showInheritedPolicies ? "Hide" : "Show"
                } inherited policies`}
              </Button>
            </span>
          )}
        {!!selectedTeamId &&
          !!teamPolicies?.length &&
          !!globalPolicies?.length &&
          showInheritedPolicies && (
            <div className={`${baseClass}__inherited-policies-table`}>
              <PoliciesListWrapper
                isLoading={isLoadingGlobalPolicies}
                policiesList={globalPolicies}
                onRemovePoliciesClick={noop}
                toggleAddPolicyModal={noop}
                resultsTitle="policies"
                resultsHtml={INHERITED_POLICIES_COUNT_HTML}
                selectedTeamId={null}
                showSelectionColumn
                tableType="inheritedPolicies"
              />
            </div>
          )}
        {showAddPolicyModal && (
          <AddPolicyModal
            onCancel={toggleAddPolicyModal}
            onSubmit={onAddPolicySubmit}
            allQueries={fleetQueries}
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
