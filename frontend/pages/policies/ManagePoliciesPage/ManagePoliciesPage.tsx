import React, { useState, useCallback, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";

// @ts-ignore
import { IConfig } from "interfaces/config";
import { IQuery } from "interfaces/query";
import { IPolicy } from "interfaces/policy";

import configAPI from "services/entities/config";
import policiesAPI from "services/entities/policies";
// @ts-ignore
import queryActions from "redux/nodes/entities/queries/actions";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import { inMilliseconds, secondsToHms } from "fleet/helpers";

import TableDataError from "components/TableDataError";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import PoliciesListWrapper from "./components/PoliciesListWrapper";
import AddPolicyModal from "./components/AddPolicyModal";
import RemovePoliciesModal from "./components/RemovePoliciesModal";

const baseClass = "manage-policies-page";

const DOCS_LINK =
  "https://github.com/fleetdm/fleet/blob/fleet-v4.3.0/docs/2-Deploying/2-Configuration.md#osquery_detail_update_interval";
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

  return (
    <PoliciesListWrapper
      policiesList={policiesList}
      isLoading={isLoading}
      onRemovePoliciesClick={onRemovePoliciesClick}
      toggleAddPolicyModal={toggleAddPolicyModal}
    />
  );
};

const ManagePolicyPage = (): JSX.Element => {
  const dispatch = useDispatch();

  const queries = useSelector((state: IRootState) => state.entities.queries);
  const queriesList = Object.values(queries.data);

  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showRemovePoliciesModal, setShowRemovePoliciesModal] = useState(false);
  const [selectedIds, setSelectedIds] = useState<number[] | never[]>([]);

  const [updateInterval, setUpdateInterval] = useState<string>(
    "update interval"
  );
  const [policies, setPolicies] = useState<IPolicy[] | never[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingError, setIsLoadingError] = useState(false);

  const getPolicies = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await policiesAPI.loadAll();
      setPolicies(response.policies);
    } catch (error) {
      console.log(error);
      setIsLoadingError(true);
    } finally {
      setIsLoading(false);
    }
  }, [dispatch]);

  const getInterval = useCallback(async () => {
    try {
      const response = await configAPI.loadAll();
      const interval = secondsToHms(
        inMilliseconds(response.update_interval.osquery_detail) / 1000
      );
      setUpdateInterval(interval);
    } catch (error) {
      console.log(error);
      dispatch(
        renderFlash(
          "error",
          "Sorry, we could not retrieve your update interval."
        )
      );
    }
  }, [dispatch]);

  useEffect(() => {
    getPolicies();
    getInterval();
  }, [getInterval, getPolicies]);

  useEffect(() => {
    dispatch(queryActions.loadAll());
  }, [dispatch]);

  const toggleAddPolicyModal = useCallback(() => {
    setShowAddPolicyModal(!showAddPolicyModal);
  }, [showAddPolicyModal, setShowAddPolicyModal]);

  const toggleRemovePoliciesModal = useCallback(() => {
    setShowRemovePoliciesModal(!showRemovePoliciesModal);
  }, [showRemovePoliciesModal, setShowRemovePoliciesModal]);

  // TODO typing for mouse event?
  const onRemovePoliciesClick = useCallback(
    (selectedTableIds: number[]): void => {
      toggleRemovePoliciesModal();
      setSelectedIds(selectedTableIds);
    },
    [toggleRemovePoliciesModal]
  );

  const onRemovePoliciesSubmit = useCallback(() => {
    const ids = selectedIds;
    policiesAPI
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
  }, [dispatch, getPolicies, selectedIds, toggleRemovePoliciesModal]);

  const onAddPolicySubmit = useCallback(
    (query_id: number | undefined) => {
      if (!query_id) {
        dispatch(
          renderFlash("error", "Could not add policy. Please try again.")
        );
        console.log("Missing query id; cannot add policy");
        return false;
      }
      policiesAPI
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

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Policies</span>
              </h1>
            </div>
          </div>
          {policies && policies.length !== 0 && !isLoadingError && (
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
        {policies && policies.length !== 0 && !isLoadingError && (
          <InfoBanner className={`${baseClass}__sandbox-info`}>
            <p>
              Your policies are checked every <b>{updateInterval.trim()}</b>.
              Check out the Fleet documentation on{" "}
              <a href={DOCS_LINK} target="_blank" rel="noreferrer">
                <b>how to edit this frequency</b>
              </a>
              .
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
            allQueries={queriesList}
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
