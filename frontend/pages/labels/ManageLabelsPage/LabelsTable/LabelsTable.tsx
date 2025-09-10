import { ILabel } from "interfaces/label";
import React from "react";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";

import { generateDataSet, generateTableHeaders } from "./LabelsTableConfig";

const baseClass = "labels-table";

interface ILabelsTable {
  labels: ILabel[];
  onClickAction: (action: string, label: ILabel) => void;
}

const LabelsTable = ({ labels, onClickAction }: ILabelsTable) => {
  const tableHeaders = generateTableHeaders(onClickAction);
  // const tableData = labels ? generateDataSet(labels, user) : [];

  const renderLabelCount = () => {
    if (!labels || labels.length === 0) {
      return <></>;
    }

    return <TableCount name="labels" count={labels.length} />;
  };

  return (
    <TableContainer
      isLoading={false}
      columnConfigs={tableHeaders}
      // data={tableData}
      data={labels}
      defaultSortHeader="name"
      defaultSortDirection="asc"
      resultsTitle="labels"
      showMarkAllPages={false}
      isAllPagesSelected={false}
      isClientSidePagination
      renderCount={renderLabelCount}
      emptyComponent={() => (
        <div className={`${baseClass}__empty-state`}>
          <p>No labels found.</p>
        </div>
      )}
    />
  );
};

export default LabelsTable;
