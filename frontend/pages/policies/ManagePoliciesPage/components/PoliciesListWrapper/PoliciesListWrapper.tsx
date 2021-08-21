import React from "react";

import Button from "components/buttons/Button";
import { IPolicy } from "interfaces/policy";
import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./PoliciesTableConfig";
// @ts-ignore
// import policiesSvg from "../../../../../../assets/images/policies.svg";

const baseClass = "policies-list-wrapper";
const noPoliciesClass = "no-policies";

interface IPoliciesListWrapperProps {
  policiesList: IPolicy[];
  isLoading: boolean;
  onRemovePoliciesClick: any;
  toggleAddPolicyModal: () => void;
}

const PoliciesListWrapper = (props: IPoliciesListWrapperProps): JSX.Element => {
  const {
    policiesList,
    isLoading,
    onRemovePoliciesClick,
    toggleAddPolicyModal,
  } = props;

  const NoPolicies = () => {
    return (
      <div className={`${noPoliciesClass}`}>
        <div className={`${noPoliciesClass}__inner`}>
          {/* <img src={policiesSvg} alt="No Policies" /> */}
          <div className={`${noPoliciesClass}__inner-text`}>
            <h2>You don&apos;t have any policies.</h2>
            <div className={`${noPoliciesClass}__-cta-buttons`}>
              <Button
                variant="brand"
                className={`${noPoliciesClass}__add-policy-button`}
                onClick={toggleAddPolicyModal}
              >
                Add a policy
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  };

  const tableHeaders = generateTableHeaders();

  return (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={generateDataSet(policiesList)}
        isLoading={isLoading}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
        onPrimarySelectActionClick={onRemovePoliciesClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="close"
        primarySelectActionButtonText={"Remove"}
        emptyComponent={NoPolicies}
      />
    </div>
  );
};

export default PoliciesListWrapper;
