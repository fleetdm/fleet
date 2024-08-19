import React from "react";
import { noop } from "lodash";

import { IPolicyHostResponse } from "interfaces/host";
import TableContainer from "components/TableContainer";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PolicyResultsTableConfig";

// TODO - this class is duplicated and styles are overlapping with PolicyErrorsTable. Differentiate
// them clearly and encapsulate common styles.
const baseClass = "policy-results-table";

interface IPolicyResultsTableProps {
  hostResponses: IPolicyHostResponse[];
  isLoading: boolean;
  resultsTitle?: string;
  canAddOrDeletePolicy?: boolean;
}

const PolicyResultsTable = ({
  hostResponses,
  isLoading,
  resultsTitle,
  canAddOrDeletePolicy,
}: IPolicyResultsTableProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle={resultsTitle || "policies"}
        columnConfigs={generateTableHeaders()}
        data={generateDataSet(hostResponses)}
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

export default PolicyResultsTable;
