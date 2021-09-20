import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery, useMutation } from "react-query";
import { Params } from "react-router/lib/Router";

import PATHS from "router/paths"; // @ts-ignore

import { useDispatch, useSelector } from "react-redux";

// @ts-ignore
import { IConfig } from "interfaces/config";
import { IPolicy } from "interfaces/policy";
import { IQuery } from "interfaces/query";
import { ITeam } from "interfaces/team";

import { AppContext } from "context/app";

// import configAPI from "services/entities/config";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";

import fleetQueriesAPI from "services/entities/queries";
import teamsAPI from "services/entities/teams";

// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import { inMilliseconds, secondsToHms } from "fleet/helpers";

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
interface IRootState {
  app: {
    config: IConfig;
  };
  entities: {
    queries: {
      isLoading: boolean;
      data: IQuery[];
    };
  };
}

const renderTable = (
  policiesList: IPolicy[],
  isLoading: boolean,
  isLoadingError: boolean,
  onRemovePoliciesClick: (selectedTableIds: number[]) => void,
  toggleAddPolicyModal: () => void
): JSX.Element => {
  if (isLoadingError) {
    return <TableDataError />;
  }
  console.log("rendering table");

  return (
    <PoliciesListWrapper
      policiesList={policiesList}
      isLoading={isLoading}
      onRemovePoliciesClick={onRemovePoliciesClick}
      toggleAddPolicyModal={toggleAddPolicyModal}
    />
  );
};

const ManagePolicyPage = (managePoliciesPageProps: {
  router: any;
  params: Params;
  location: any;
}): JSX.Element => {
  const { location, params, router } = managePoliciesPageProps;
  const dispatch = useDispatch();

  const {
    config,
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
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
    // enabled: !!isPremiumTier,
    select: (data) => data.queries,
  });

  const [userTeams, setUserTeams] = useState<ITeam[] | never[]>([]);

  const [policyTeamId, setPolicyTeamId] = useState<number | null>(null);
  const [policyQueryIds, setPolicyQueryIds] = useState<number[] | never[]>([]);

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
      router.push(`${PATHS.MANAGE_POLICIES}?team_id=${id}`);
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
      setPolicyQueryIds(selectedTableIds);
    },
    [toggleRemovePoliciesModal]
  );

  const onRemovePoliciesSubmit = useCallback(() => {
    const ids = policyQueryIds;
    globalPoliciesAPI
      .destroy(ids)
      .then(() => {
        dispatch(
          renderFlash(
            "success",
            `Successfully removed ${
              ids && ids.length === 1 ? "policy" : "policies"
            }.`
          )
        );
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Unable to remove ${
              ids && ids.length === 1 ? "policy" : "policies"
            }. Please try again.`
          )
        );
      })
      .finally(() => {
        toggleRemovePoliciesModal();
        getPolicies();
      });
  }, [dispatch, getPolicies, policyQueryIds, toggleRemovePoliciesModal]);

  const onAddPolicySubmit = useCallback(
    (query_id: number | undefined) => {
      if (!query_id) {
        dispatch(
          renderFlash("error", "Could not add policy. Please try again.")
        );
        console.log("Missing query id; cannot add policy");
        return false;
      }
      globalPoliciesAPI
        .create(query_id)
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
          getPolicies();
        });
      return false;
    },
    [dispatch, getPolicies, toggleAddPolicyModal]
  );

  // Parse url query params and set selected team
  useEffect(() => {
    const { team_id } = location.query;
    const teamId = parseInt(team_id, 10) || 0;
    setPolicyTeamId(teamId);
  }, [location]);

  useEffect(() => {
    if (policyTeamId !== null) {
      getPolicies(policyTeamId);
    }
  }, [getPolicies, policyTeamId]);

  // Set list of teams that the current user has permission to view
  useEffect(() => {
    if (isOnGlobalTeam) {
      setUserTeams(teams);
    } else if (currentUser && currentUser.teams) {
      setUserTeams(currentUser.teams);
    }
  }, [currentUser, isOnGlobalTeam, teams]);

  // Set the update interval value that will be displayed in the InfoBanner element
  useEffect(() => {
    console.log("config ", config); // why is context config incomplete and missing update interval?
    const interval = config?.update_interval?.osquery_detail;
    if (interval) {
      setUpdateInterval(secondsToHms(inMilliseconds(interval) / 1000));
    }
  }, [config]);

  console.log(location);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                {isPremiumTier && policyTeamId !== null ? (
                  <TeamsDropdown
                    currentUserTeams={userTeams}
                    onChange={handleChangeSelectedTeam}
                    selectedTeam={policyTeamId}
                  />
                ) : (
                  <span>Policies</span>
                )}
              </h1>
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
          <p>Policy queries report which hosts are compliant.</p>
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
          {renderTable(
            policies,
            isLoading,
            isLoadingError,
            onRemovePoliciesClick,
            toggleAddPolicyModal
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
