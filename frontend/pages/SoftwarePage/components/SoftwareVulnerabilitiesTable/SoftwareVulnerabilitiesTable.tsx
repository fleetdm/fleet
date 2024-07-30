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
import { CONTACT_FLEET_LINK, GITHUB_NEW_ISSUE_LINK } from "utilities/constants";
import { DisplayPlatform } from "interfaces/platform";
import { buildQueryStringFromParams } from "utilities/url";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import generateTableConfig from "./SoftwareVulnerabilitiesTableConfig";

const baseClass = "software-vulnerabilities-table";

interface INoVulnsDetectedProps {
  itemName: string;
}

interface IVulnsNotSupportedProps {
  platformText?: DisplayPlatform;
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

export const VulnsNotSupported = ({
  platformText,
}: IVulnsNotSupportedProps) => (
  <EmptyTable
    header="Vulnerabilities are not supported for this type of host"
    info={
      <>
        Interested in vulnerabilities in {platformText ?? "this platform"}?{" "}
        <CustomLink url={CONTACT_FLEET_LINK} text="Let us know" newTab />
      </>
    }
  />
);

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

  const renderVulnerabilitiesCount = () => (
    <TableCount name="items" count={data?.length} />
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
        resultsTitle="items"
        showMarkAllPages={false}
        disableMultiRowSelect
        onSelectSingleRow={handleRowSelect}
        disableTableHeader={data.length === 0}
        renderCount={renderVulnerabilitiesCount}
      />
    </div>
  );
};

export default SoftwareVulnerabilitiesTable;
