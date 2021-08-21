import React, { useState, useCallback, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";

// @ts-ignore
import { IConfig } from "interfaces/config";
import { IQuery } from "interfaces/query";
import { IPolicy } from "interfaces/policy";
// @ts-ignore
import policiesAPI from "services/entities/policies";
// @ts-ignore
import queryActions from "redux/nodes/entities/queries/actions";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import Button from "components/buttons/Button";
// @ts-ignore
import PolicyError from "./components/PolicyError";
import PoliciesListWrapper from "./components/PoliciesListWrapper";
import AddPolicyModal from "./components/AddPolicyModal";
import RemovePoliciesModal from "./components/RemovePoliciesModal";

const baseClass = "manage-policies-page";

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
  isLoadingError: boolean,
  isLoading: boolean,
  onRemovePoliciesClick: React.MouseEventHandler<HTMLButtonElement>,
  toggleAddPolicyModal: () => void
): JSX.Element => {
  if (isLoadingError) {
    return <PolicyError />;
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

  const [showAddPolicyModal, setShowAddPolicyModal] = useState(false);
  const [showRemovePoliciesModal, setShowRemovePoliciesModal] = useState(false);
  const [selectedIds, setSelectedIds] = useState<number[] | never[]>([]);

  const [policies, setPolicies] = useState<IPolicy[] | never[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingError, setIsLoadingError] = useState(false);

  const getPolicies = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await policiesAPI.loadAll();
      console.log(response);
      setPolicies(response.policies);
    } catch (error) {
      console.log(error);
      dispatch(
        renderFlash("error", "Sorry, we could not retrieve your policies.")
      );
      setIsLoadingError(true);
    } finally {
      setIsLoading(false);
    }
  }, [dispatch]);

  useEffect(() => {
    getPolicies();
  }, [getPolicies]);

  useEffect(() => {
    dispatch(queryActions.loadAll());
  }, [dispatch]);

  const allQueries = useSelector((state: IRootState) => state.entities.queries);
  const allQueriesList = Object.values(allQueries.data);

  const toggleAddPolicyModal = useCallback(() => {
    setShowAddPolicyModal(!showAddPolicyModal);
  }, [showAddPolicyModal, setShowAddPolicyModal]);

  const toggleRemovePoliciesModal = useCallback(() => {
    setShowRemovePoliciesModal(!showRemovePoliciesModal);
  }, [showRemovePoliciesModal, setShowRemovePoliciesModal]);

  // TODO does this need to be a callback?
  // TODO typing for mouse event?
  const onRemovePoliciesClick = useCallback(
    (selectedTableIds: any): void => {
      toggleRemovePoliciesModal();
      setSelectedIds(selectedTableIds);
    },
    [toggleRemovePoliciesModal]
  );

  const onRemovePoliciesSubmit = useCallback(
    (ids: number[]) => {
      policiesAPI
        .destroy(ids)
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              `Successfully removed ${
                ids.length === 1 ? "policy" : "policies"
              }.`
            )
          );
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Unable to remove ${
                ids.length === 1 ? "policy" : "policies"
              }. Please try again.`
            )
          );
        })
        .finally(() => {
          toggleRemovePoliciesModal();
          getPolicies();
        });
    },
    [dispatch, getPolicies, toggleRemovePoliciesModal]
  );

  const onAddPolicySubmit = useCallback(
    (query_id: number) => {
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
              <div className={`${baseClass}__description`}>
                <p>Policy queries report which hosts are compliant.</p>
              </div>
            </div>
          </div>
          {/* Hide CTA Buttons if no policy or policy error */}
          {policies.length !== 0 && isLoadingError && (
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
            allQueries={allQueriesList}
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
