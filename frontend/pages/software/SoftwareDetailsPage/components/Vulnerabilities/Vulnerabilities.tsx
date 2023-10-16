import React, { useMemo } from "react";

import { ISoftware } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";

import generateVulnTableHeaders from "./VulnTableConfig";

const baseClass = "vulnerabilities";

interface IVulnerabilitiesProps {
  isLoading: boolean;
  isPremiumTier: boolean;
  isSandboxMode?: boolean;
  software: ISoftware;
}

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

const Vulnerabilities = ({
  isLoading,
  isPremiumTier,
  isSandboxMode = false,
  software,
}: IVulnerabilitiesProps): JSX.Element => {
  const tableHeaders = useMemo(
    () => generateVulnTableHeaders(isPremiumTier, isSandboxMode),
    [isPremiumTier, isSandboxMode]
  );

  return (
    <div className="section section--vulnerabilities">
      <p className="section__header">Vulnerabilities</p>
      {software?.vulnerabilities?.length ? (
        <>
          {software && (
            <div className="vuln-table">
              <TableContainer
                columns={tableHeaders}
                data={software.vulnerabilities}
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
          )}
        </>
      ) : (
        <NoVulnsDetected />
      )}
    </div>
  );
};
export default Vulnerabilities;
