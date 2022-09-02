import React from "react";

import { IMunkiIssue } from "interfaces/host";
import TableContainer from "components/TableContainer";

import EmptyState from "../EmptyState";

import { munkiIssuesTableHeaders } from "./MunkiIssuesTableConfig";

const baseClass = "munki-issues";

interface IMunkiIssuesTableProps {
  isLoading: boolean;
  munkiIssues?: IMunkiIssue[];
  deviceType?: string;
}

const MunkiIssuesTable = ({
  isLoading,
  munkiIssues,
  deviceType,
}: IMunkiIssuesTableProps): JSX.Element => {
  const tableMunkiIssues = munkiIssues;
  const tableHeaders = munkiIssuesTableHeaders;

  const EmptyMunkiIssues = () => {
    return (
      <div className={`section section--${baseClass}`}>
        <p className="section__header">Munki issues</p>
        <EmptyState title="munki-issues" reason="none-detected" />
      </div>
    );
  };

  return (
    <div className={`section section--${baseClass}`}>
      <p className="section__header">Munki issues</p>

      {munkiIssues?.length ? (
        <div className={deviceType || ""}>
          <TableContainer
            columns={tableHeaders}
            data={tableMunkiIssues || []}
            isLoading={isLoading}
            defaultSortHeader={"name"}
            defaultSortDirection={"asc"}
            resultsTitle={"issue"}
            emptyComponent={EmptyMunkiIssues}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            isClientSidePagination
            highlightOnHover
          />
        </div>
      ) : (
        <EmptyState title="munki-issues" reason="none-detected" />
      )}
    </div>
  );
};
export default MunkiIssuesTable;
