import React from "react";
import { noop } from "lodash";

import { IHostPolicyQueryError } from "interfaces/host";
import TableContainer from "components/TableContainer";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PolicyQueriesErrorsTableConfig";

const baseClass = "policies-queries-list-wrapper";
const noPolicyQueries = "no-policy-queries";

interface IPoliciesListWrapperProps {
  errorsList: IHostPolicyQueryError[];
  isLoading: boolean;
  resultsTitle?: string;
  canAddOrRemovePolicy?: boolean;
}

const PoliciesListWrapper = ({
  errorsList,
  isLoading,
  resultsTitle,
  canAddOrRemovePolicy,
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
        canAddOrRemovePolicy ? "" : "hide-selection-column"
      }`}
    >
      <TableContainer
        resultsTitle={resultsTitle || "policies"}
        columns={generateTableHeaders()}
        data={generateDataSet(errorsList)}
        isLoading={isLoading}
        defaultSortHeader={"name"}
        defaultSortDirection={"asc"}
        manualSortBy
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
