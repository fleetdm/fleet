import React, { useMemo } from "react";

import { IMunkiIssue } from "interfaces/host";
import TableContainer from "components/TableContainer";

import EmptyState from "../EmptyState";

import {
  generateMunkiIssuesTableHeaders,
  generateMunkiIssuesTableData,
} from "./MunkiIssuesTableConfig";

const baseClass = "host-details";

interface IMunkiIssuesTableProps {
  isLoading: boolean;
  munkiIssues?: IMunkiIssue[];
  deviceUser?: boolean;
  deviceType?: string;
}

const MunkiIssuesTable = ({
  isLoading,
  munkiIssues,
  deviceUser,
  deviceType,
}: IMunkiIssuesTableProps): JSX.Element => {
  const tableMunkiIssues = useMemo(
    () => generateMunkiIssuesTableData(munkiIssues),
    [munkiIssues]
  );
  const tableHeaders = useMemo(
    () => generateMunkiIssuesTableHeaders(deviceUser),
    [deviceUser]
  );

  const EmptyMunkiIssues = () => {
    return (
      <div className="section section--munki-issues">
        <p className="section__header">Munki issues</p>
        <EmptyState title="munki-issues" reason="none-detected" />
      </div>
    );
  };

  return (
    <div className="section section--munki-issues">
      <p className="section__header">Munki issues</p>

      {munkiIssues?.length ? (
        <div className={deviceType || ""}>
          <TableContainer
            columns={tableHeaders}
            data={tableMunkiIssues || []}
            isLoading={isLoading}
            defaultSortHeader={"name"}
            defaultSortDirection={"asc"}
            resultsTitle={"issues"}
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
