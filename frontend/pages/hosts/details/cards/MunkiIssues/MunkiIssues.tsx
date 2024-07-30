import React from "react";

import { IMunkiIssue } from "interfaces/host";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import Card from "components/Card";

import { munkiIssuesTableHeaders } from "./MunkiIssuesTableConfig";

const baseClass = "munki-issues-card";

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

  return (
    <Card
      className={`${baseClass} card`}
      borderRadiusSize="xxlarge"
      includeShadow
      largePadding
    >
      <p className="card__header">Munki issues</p>

      {munkiIssues?.length ? (
        <div className={deviceType || ""}>
          <TableContainer
            columnConfigs={tableHeaders}
            data={tableMunkiIssues || []}
            isLoading={isLoading}
            defaultSortHeader="name"
            defaultSortDirection="asc"
            resultsTitle="issue"
            emptyComponent={() => (
              <EmptyTable
                header="No Munki issues detected"
                info="The last time Munki ran on this host, no issues were reported."
              />
            )}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            isClientSidePagination
          />
        </div>
      ) : (
        <EmptyTable
          header="No Munki issues detected"
          info="The last time Munki ran on this host, no issues were reported."
        />
      )}
    </Card>
  );
};
export default MunkiIssuesTable;
