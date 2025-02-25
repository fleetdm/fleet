import React from "react";
import { noop } from "lodash";

import TableContainer from "components/TableContainer";
import { ICampaignError } from "interfaces/campaign";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PolicyErrorsTableConfig";

// TODO - this class is duplicated and styles are overlapping with PolicyResultsTable. Differentiate
// them clearly and encapsulate common styles.
const baseClass = "policy-results-table";

interface IPolicyErrorsTableProps {
  errorsList: ICampaignError[];
  isLoading: boolean;
  resultsTitle?: string;
  canAddOrDeletePolicy?: boolean;
}

const PolicyErrorsTable = ({
  errorsList,
  isLoading,
  resultsTitle,
  canAddOrDeletePolicy,
}: IPolicyErrorsTableProps): JSX.Element => {
  return (
    <div className={baseClass}>
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
        emptyComponent={() => (
          <div className="no-hosts__inner">
            <p>No hosts are online.</p>
          </div>
        )}
        onQueryChange={noop}
        disableCount
      />
    </div>
  );
};

export default PolicyErrorsTable;
