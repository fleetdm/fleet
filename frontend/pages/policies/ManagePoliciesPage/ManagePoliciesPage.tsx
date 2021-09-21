import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import { useDispatch } from "react-redux";
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

const ManagePolicyPage = (managePoliciesPageProps: {
  router: any;
  location: any;
}): JSX.Element => {
  const { location, router } = managePoliciesPageProps;
  const dispatch = useDispatch();

  const {
    config,
    currentUser,
    isGlobalAdmin,
    isOnGlobalTeam,
    isOnlyObserver,
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

  const [userTeams, setUserTeams] = useState<ITeam[] | never[]>([]);
  const [selectedTeamId, setSelectedTeamId] = useState<number | null>(null);
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<
    number[] | never[]
  >([]);

  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showRemovePoliciesModal, setShowRemovePoliciesModal] = useState(false);

  const [updateInterval, setUpdateInterval] = useState<string>(
    "update interval"
  );

  const [policies, setPolicies] = useState<IPolicy[] | never[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingError, setIsLoadingError] = useState(false);

  const getPolicies = useCallback(async (teamId = 0) => {
    setIsLoading(true);
    setIsLoadingError(false);
    try {
      const response = teamId
        ? await teamPoliciesAPI.loadAll(teamId)
        : await globalPoliciesAPI.loadAll();
      setPolicies(response.policies);
    } catch (error) {
      console.log(error);
      setIsLoadingError(true);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const handleChangeSelectedTeam = useCallback(
    (id: number) => {
      const { MANAGE_POLICIES } = PATHS;
      const path = id ? `${MANAGE_POLICIES}?team_id=${id}` : MANAGE_POLICIES;
      router.replace(path);
    },
    [router]
  );

  const toggleAddPolicyModal = useCallback(() => {
    setShowAddPolicyModal(!showAddPolicyModal);
  }, [showAddPolicyModal, setShowAddPolicyModal]);

  const toggleRemovePoliciesModal = useCallback(() => {
    setShowRemovePoliciesModal(!showRemovePoliciesModal);
  }, [showRemovePoliciesModal, setShowRemovePoliciesModal]);

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
        getPolicies();
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

  // Parse url query params and set selected team
  useEffect(() => {
    const { team_id: param } = location.query;
    let teamId: number | null = parseInt(param, 10) || 0;
    if (!userTeams.find((t) => t.id === teamId)) {
      if (isOnGlobalTeam) {
        teamId !== 0 && handleChangeSelectedTeam(0);
        teamId = 0;
      } else {
        teamId = userTeams[0]?.id || null;
        teamId && handleChangeSelectedTeam(teamId);
      }
    }
    teamId !== null && setSelectedTeamId(teamId);
  }, [handleChangeSelectedTeam, isOnGlobalTeam, location, userTeams]);

  useEffect(() => {
    if (selectedTeamId !== null) {
      getPolicies(selectedTeamId);
    }
  }, [getPolicies, selectedTeamId]);

  // Set list of teams that the current user has permission to view
  useEffect(() => {
    const unsortedTeams = [];
    if (isOnGlobalTeam && teams) {
      unsortedTeams.push(...teams);
    } else if (currentUser && currentUser.teams) {
      unsortedTeams.push(...currentUser.teams);
    }
    setUserTeams(
      unsortedTeams.sort((a, b) => sortUtils.caseInsensitiveAsc(b.name, a.name))
    );
  }, [currentUser, isOnGlobalTeam, teams]);

  // Set the update interval value that will be displayed in the InfoBanner element
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
                {isPremiumTier && selectedTeamId !== null ? (
                  <TeamsDropdown
                    currentUserTeams={userTeams}
                    onChange={handleChangeSelectedTeam}
                    selectedTeam={selectedTeamId}
                  />
                ) : (
                  <h1>Policies</h1>
                )}
              </div>
            </div>
          </div>
          {!isOnlyObserver && (
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
          {!isPremiumTier || !selectedTeamId ? (
            <p>
              Add policies for <b>all of your hosts</b> to see which pass your
              organization’s standards.{" "}
            </p>
          ) : (
            <p>
              Add additional policies for <b>all hosts assigned to this team</b>
              .
            </p>
          )}
        </div>
        {config && updateInterval && (
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
          {isLoadingError ? (
            <TableDataError />
          ) : (
            <PoliciesListWrapper
              policiesList={policies}
              selectedTeamId={selectedTeamId}
              isLoading={isLoading}
              onRemovePoliciesClick={onRemovePoliciesClick}
              toggleAddPolicyModal={toggleAddPolicyModal}
            />
          )}
        </div>
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
