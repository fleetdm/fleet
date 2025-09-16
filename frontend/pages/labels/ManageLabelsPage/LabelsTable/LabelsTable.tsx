import { ILabel } from "interfaces/label";
import React from "react";

import { IUser } from "interfaces/user";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";

import { generateDataSet, generateTableHeaders } from "./LabelsTableConfig";

const baseClass = "labels-table";

interface ILabelsTable {
  labels: ILabel[];
  onClickAction: (action: string, label: ILabel) => void;
  currentUser: IUser;
}

const LabelsTable = ({ labels, onClickAction, currentUser }: ILabelsTable) => {
  const tableHeaders = generateTableHeaders(currentUser, onClickAction);

  const tableData = generateDataSet(labels);

  return (
    <TableContainer
      isLoading={false}
      columnConfigs={tableHeaders}
      data={tableData}
      defaultSortHeader="name"
      defaultSortDirection="asc"
      resultsTitle="labels"
      showMarkAllPages={false}
      isAllPagesSelected={false}
      isClientSidePagination
      renderCount={() => <TableCount name="labels" count={tableData.length} />}
      emptyComponent={() => (
        <div className={`${baseClass}__empty-state`}>
          <p>No labels found.</p>
        </div>
      )}
    />
  );
};

export default LabelsTable;
