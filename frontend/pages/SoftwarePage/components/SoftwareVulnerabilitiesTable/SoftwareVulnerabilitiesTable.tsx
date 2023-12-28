import React, { useContext, useMemo } from "react";
import classnames from "classnames";

import { AppContext } from "context/app";
import { ISoftwareVulnerability } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import generateTableConfig from "./SoftwareVulnerabilitiesTableConfig";

const baseClass = "software-vulnerabilities-table";

const NoVulnsDetected = (): JSX.Element => {
  return (
    <EmptyTable
      header="No vulnerabilities detected for this software item."
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
  isLoading: boolean;
  className?: string;
}

const SoftwareVulnerabilitiesTable = ({
  data,
  isLoading,
  className,
}: ISoftwareVulnerabilitiesTableProps) => {
  const { isPremiumTier, isSandboxMode } = useContext(AppContext);

  const classNames = classnames(baseClass, className);

  const tableHeaders = useMemo(
    () => generateTableConfig(Boolean(isPremiumTier), Boolean(isSandboxMode)),
    [isPremiumTier, isSandboxMode]
  );
  return (
    <div className={classNames}>
      <TableContainer
        columnConfigs={tableHeaders}
        data={data}
        defaultSortHeader={isPremiumTier ? "epss_probability" : "cve"}
        defaultSortDirection={"desc"}
        emptyComponent={NoVulnsDetected}
        isAllPagesSelected={false}
        isLoading={isLoading}
        isClientSidePagination
        pageSize={20}
        resultsTitle={"vulnerabilities"}
        showMarkAllPages={false}
      />
    </div>
  );
};

export default SoftwareVulnerabilitiesTable;
