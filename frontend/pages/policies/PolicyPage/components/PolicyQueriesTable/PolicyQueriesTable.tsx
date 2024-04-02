import React from "react";
import { noop } from "lodash";

import { IHostPolicyQuery } from "interfaces/host";
import TableContainer from "components/TableContainer";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PolicyQueriesTableConfig";

const baseClass = "policies-queries-table";
const noPolicyQueries = "no-policy-queries";

interface IPoliciesTableProps {
  policyHostsList: IHostPolicyQuery[];
  isLoading: boolean;
  resultsTitle?: string;
  canAddOrDeletePolicy?: boolean;
}

const PoliciesTable = ({
  policyHostsList,
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
        data={generateDataSet(policyHostsList)}
        isLoading={isLoading}
        defaultSortHeader="query_results"
        defaultSortDirection="asc"
        showMarkAllPages={false}
        isAllPagesSelected={false}
        isClientSidePagination
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
