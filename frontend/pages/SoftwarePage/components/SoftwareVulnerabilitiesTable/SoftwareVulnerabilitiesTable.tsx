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
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import generateTableConfig from "./SoftwareVulnerabilitiesTableConfig";

const baseClass = "software-vulnerabilities-table";

interface INoVulnsDetectedProps {
  itemName: string;
}

const NoVulnsDetected = ({ itemName }: INoVulnsDetectedProps): JSX.Element => {
  return (
    <EmptyTable
      header={`No vulnerabilities detected for this ${itemName}`}
      info={
        <>
          Expecting to see vulnerabilities?{" "}
          <CustomLink
            url={GITHUB_NEW_ISSUE_LINK}
            text="File an issue on GitHub"
            newTab
          />
        </>
      }
    />
  );
};

interface ISoftwareVulnerabilitiesTableProps {
  data: ISoftwareVulnerability[];
  /** Name displayed on the empty state */
  itemName: string;
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
  itemName,
  isLoading,
  className,
  router,
  teamIdForApi,
}: ISoftwareVulnerabilitiesTableProps) => {
  const { isPremiumTier, isSandboxMode } = useContext(AppContext);

  const classNames = classnames(baseClass, className);

  const handleRowSelect = (row: IRowProps) => {
    const hostsBySoftwareParams = {
      cve: row.original.cve,
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
    () =>
      generateTableConfig(
        Boolean(isPremiumTier),
        Boolean(isSandboxMode),
        router,
        teamIdForApi
      ),
    [isPremiumTier, isSandboxMode]
  );
  return (
    <div className={classNames}>
      <TableContainer
        columnConfigs={tableHeaders}
        data={data}
        defaultSortHeader={isPremiumTier ? "updated_at" : "cve"} // TODO: Change premium to created_at when added to API
        defaultSortDirection="desc"
        emptyComponent={() => <NoVulnsDetected itemName={itemName} />}
        isAllPagesSelected={false}
        isLoading={isLoading}
        isClientSidePagination
        pageSize={20}
        resultsTitle="vulnerabilities"
        showMarkAllPages={false}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
        disableTableHeader={data.length === 0}
      />
    </div>
  );
};

export default SoftwareVulnerabilitiesTable;
