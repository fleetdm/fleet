import React from "react";
import { noop } from "lodash";

import { IHostPolicyQuery } from "interfaces/host";
import TableContainer from "components/TableContainer";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PolicyQueriesTableConfig";

const baseClass = "policies-queries-list-wrapper";
const noPolicyQueries = "no-policy-queries";

interface IPoliciesListWrapperProps {
  policyHostsList: IHostPolicyQuery[];
  isLoading: boolean;
  resultsTitle?: string;
  canAddOrDeletePolicy?: boolean;
}

const PoliciesListWrapper = ({
  policyHostsList,
  isLoading,
  resultsTitle,
  canAddOrDeletePolicy,
}: IPoliciesListWrapperProps): JSX.Element => {
  const NoPolicyQueries = () => {
    return (
      <div className={`${noPolicyQueries}__inner`}>
        <p>No hosts are online.</p>
      </div>
    );
  };

  return (
    <div
      className={`${baseClass} ${
        canAddOrDeletePolicy ? "" : "hide-selection-column"
      }`}
    >
      <TableContainer
        resultsTitle={resultsTitle || "policies"}
        columns={generateTableHeaders()}
        data={generateDataSet(policyHostsList)}
        isLoading={isLoading}
        defaultSortHeader={"query_results"}
        defaultSortDirection={"asc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        emptyComponent={NoPolicyQueries}
        onQueryChange={noop}
        disableCount
      />
    </div>
  );
};

export default PoliciesListWrapper;
