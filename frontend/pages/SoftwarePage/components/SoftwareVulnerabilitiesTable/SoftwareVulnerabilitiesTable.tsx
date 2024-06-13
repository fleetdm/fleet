/**
software/versions/:id > Vulnerabilities table
software/os/:id > Vulnerabilities table
*/

import React, { useContext, useMemo } from "react";
import classnames from "classnames";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import { ISoftwareVulnerability } from "interfaces/software";
import { buildQueryStringFromParams } from "utilities/url";
import TableContainer from "components/TableContainer";

import generateTableConfig from "./SoftwareVulnerabilitiesTableConfig";
import EmptySoftwareTable from "../EmptySoftwareTable";

const baseClass = "software-vulnerabilities-table";

interface ISoftwareVulnerabilitiesTableProps {
  data: ISoftwareVulnerability[];
  isSoftwareEnabled?: boolean;
  itemName?: string;
  isLoading: boolean;
  className?: string;
  router: InjectedRouter;
  teamIdForApi?: number;
}

interface IRowProps extends Row {
  original: {
    cve?: string;
  };
}

const SoftwareVulnerabilitiesTable = ({
  data,
  isSoftwareEnabled,
  isLoading,
  itemName,
  className,
  router,
  teamIdForApi,
}: ISoftwareVulnerabilitiesTableProps) => {
  const { isPremiumTier } = useContext(AppContext);

  const classNames = classnames(baseClass, className);

  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = {
      vulnerability: row.original.cve,
      team_id: teamIdForApi,
    };

    const path = hostsBySoftwareParams
      ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(
          hostsBySoftwareParams
        )}`
      : PATHS.MANAGE_HOSTS;

    router.push(path);
  };

  const tableHeaders = useMemo(
    () => generateTableConfig(Boolean(isPremiumTier), router, teamIdForApi),
    [isPremiumTier]
  );
  return (
    <div className={classNames}>
      <TableContainer
        columnConfigs={tableHeaders}
        data={data}
        defaultSortHeader={isPremiumTier ? "updated_at" : "cve"} // TODO: Change premium to created_at when added to API
        defaultSortDirection="desc"
        emptyComponent={() => (
          <EmptySoftwareTable
            tableName="vulnerabilities"
            isSoftwareDisabled={!isSoftwareEnabled}
          />
        )}
        isAllPagesSelected={false}
        isLoading={isLoading}
        isClientSidePagination
        pageSize={20}
        resultsTitle="items"
        showMarkAllPages={false}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
        disableTableHeader={data.length === 0}
      />
    </div>
  );
};

export default SoftwareVulnerabilitiesTable;
