import React from "react";

import { ISoftware } from "interfaces/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import TableContainer from "components/TableContainer";

import generateVulnTableHeaders from "./VulnTableConfig";
import ExternalLinkIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "vulnerabilities";

interface ISoftwareTableProps {
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
            File an issue on GitHub{" "}
            <img alt="External link" src={ExternalLinkIcon} />
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
}: ISoftwareTableProps): JSX.Element => {
  const tableHeaders = generateVulnTableHeaders(isPremiumTier);

  const vulns = software.vulnerabilities?.map((v) => ({
    ...v,
    cisa_known_exploit: Math.random() > 0.5,
  }));

  console.log(vulns);

  return (
    <div className="section section--vulnerabilities">
      <p className="section__header">Vulnerabilities</p>

      {software?.vulnerabilities?.length ? (
        <>
          {software && (
            <div className="vuln-table">
              <TableContainer
                columns={tableHeaders}
                // data={software.vulnerabilities}
                data={vulns
                  ?.concat(vulns)
                  .concat(vulns)
                  .concat(vulns)
                  .concat(vulns)}
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
