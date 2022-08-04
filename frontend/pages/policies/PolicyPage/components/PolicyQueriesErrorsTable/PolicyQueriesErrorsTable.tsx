import React from "react";
import { noop } from "lodash";

import TableContainer from "components/TableContainer";
import { ICampaignError } from "interfaces/campaign";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PolicyQueriesErrorsTableConfig";

const baseClass = "policies-queries-table";
const noPolicyQueries = "no-policy-queries";

interface IPoliciesTableProps {
  errorsList: ICampaignError[];
  isLoading: boolean;
  resultsTitle?: string;
  canAddOrDeletePolicy?: boolean;
}

const PoliciesTable = ({
  errorsList,
  isLoading,
  resultsTitle,
  canAddOrDeletePolicy,
}: IPoliciesTableProps): JSX.Element => {
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

export default PoliciesTable;
