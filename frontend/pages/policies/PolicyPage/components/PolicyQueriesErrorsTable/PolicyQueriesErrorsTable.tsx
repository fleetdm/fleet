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
        columnConfigs={generateTableHeaders()}
        data={generateDataSet(errorsList)}
        isLoading={isLoading}
        defaultSortHeader="name"
        defaultSortDirection="asc"
        manualSortBy
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
        primarySelectAction={{
          name: "delete policy",
          buttonText: "Delete",
          iconSvg: "trash",
          variant: "text-icon",
        }}
        emptyComponent={NoPolicyQueries}
        onQueryChange={noop}
        disableCount
      />
    </div>
  );
};

export default PoliciesTable;
