import React, { useMemo } from "react";

import { ISoftware } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import TableContainer from "components/TableContainer";

import generateVulnTableHeaders from "./VulnTableConfig";
import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

const baseClass = "vulnerabilities";

interface IVulnerabilitiesProps {
  isLoading: boolean;
  isPremiumTier: boolean;
  software: ISoftware;
}

const NoVulnsDetected = (): JSX.Element => {
  return (
    <div className={`${baseClass}__empty-vulnerabilities`}>
      <div className="empty-vulnerabilities__inner">
        <h1>No vulnerabilities detected for this software item.</h1>
        <p>
          Expecting to see vulnerabilities?{" "}
          <a
            href={GITHUB_NEW_ISSUE_LINK}
            target="_blank"
            rel="noopener noreferrer"
          >
            File an issue on GitHub
            <img src={ExternalLinkIcon} alt="Open external link" />
          </a>
        </p>
      </div>
    </div>
  );
};

const Vulnerabilities = ({
  isLoading,
  isPremiumTier,
  software,
}: IVulnerabilitiesProps): JSX.Element => {
  const tableHeaders = useMemo(() => generateVulnTableHeaders(isPremiumTier), [
    isPremiumTier,
  ]);

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
                highlightOnHover
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
