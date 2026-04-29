import React, { memo } from "react";

import { ILabel } from "interfaces/label";

import { IUser } from "interfaces/user";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyState from "components/EmptyState";

import { generateDataSet, generateTableHeaders } from "./LabelsTableConfig";

const baseClass = "labels-table";

interface ILabelsTable {
  labels: ILabel[];
  onClickAction: (action: string, label: ILabel) => void;
  currentUser: IUser;
  labelsGitOpsManaged?: boolean;
  repoURL?: string;
}

const LabelsTable = ({
  labels,
  onClickAction,
  currentUser,
  labelsGitOpsManaged = false,
  repoURL,
}: ILabelsTable) => {
  const tableHeaders = generateTableHeaders(
    currentUser,
    onClickAction,
    labelsGitOpsManaged,
    repoURL
  );

  const tableData = generateDataSet(labels);

  return (
    <TableContainer
      className={baseClass}
      isLoading={false}
      columnConfigs={tableHeaders}
      data={tableData}
      defaultSortHeader="name"
      defaultSortDirection="asc"
      resultsTitle="labels"
      showMarkAllPages={false}
      isAllPagesSelected={false}
      isClientSidePagination
      renderCount={() =>
        tableData.length ? (
          <TableCount name="labels" count={tableData.length} />
        ) : null
      }
      emptyComponent={() => (
        <EmptyState
          header="No labels"
          info="Labels you create will appear here."
        />
      )}
    />
  );
};

export default memo(LabelsTable);
