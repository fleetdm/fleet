import React from "react";
import { noop } from "lodash";

import Button from "components/buttons/Button";
import { IPolicy } from "interfaces/policy";
import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./PoliciesTableConfig";

const baseClass = "policies-list-wrapper";
const noPoliciesClass = "no-policies";

interface IPoliciesListWrapperProps {
  policiesList: IPolicy[];
  isLoading: boolean;
  onRemovePoliciesClick: (selectedTableIds: number[]) => void;
  toggleAddPolicyModal: () => void;
  resultsTitle?: string;
  resultsHtml?: JSX.Element;
  selectedTeamId?: number | undefined | null;
  canAddOrRemovePolicy?: boolean;
  tableType?: string;
}

const PoliciesListWrapper = (props: IPoliciesListWrapperProps): JSX.Element => {
  const {
    policiesList,
    isLoading,
    onRemovePoliciesClick,
    toggleAddPolicyModal,
    resultsTitle,
    resultsHtml,
    selectedTeamId,
    canAddOrRemovePolicy,
    tableType,
  } = props;

  const NoPolicies = () => {
    return (
      <div className={`${noPoliciesClass}`}>
        <div className={`${noPoliciesClass}__inner`}>
          <div className={`${noPoliciesClass}__inner-text`}>
            <h2>You don&apos;t have any policies.</h2>
            <p>
              Policies allow you to monitor which devices meet a certain
              standard.
            </p>
            {canAddOrRemovePolicy && (
              <div className={`${noPoliciesClass}__-cta-buttons`}>
                <Button
                  variant="brand"
                  className={`${noPoliciesClass}__add-policy-button`}
                  onClick={toggleAddPolicyModal}
                >
                  Add a policy
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  };

  return (
    <div
      className={`${baseClass} ${
        canAddOrRemovePolicy ? "" : "hide-selection-column"
      }`}
    >
      <TableContainer
        resultsTitle={resultsTitle || "policies"}
        resultsHtml={resultsHtml}
        columns={generateTableHeaders({
          selectedTeamId,
          showSelectionColumn: canAddOrRemovePolicy,
          tableType,
        })}
        data={generateDataSet(policiesList)}
        isLoading={isLoading}
        defaultSortHeader={"query_name"}
        defaultSortDirection={"asc"}
        manualSortBy
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
        onPrimarySelectActionClick={onRemovePoliciesClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="close"
        primarySelectActionButtonText={"Remove"}
        emptyComponent={NoPolicies}
        onQueryChange={noop}
      />
    </div>
  );
};

export default PoliciesListWrapper;
