import React from "react";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import { IEmptyTableProps } from "interfaces/empty_table";

export interface IEmptyVulnerabilitiesTableProps {
  isPremiumTier?: boolean;
  teamId?: number;
  exploitedFilter?: boolean;
  isSoftwareDisabled?: boolean;
  searchQuery?: string;
  knownVulnerability?: boolean;
}

const isValidCVEFormat = (query: string): boolean => {
  const cveRegex = /^(CVE-)?\d{4}-\d{4,}$/i;
  return cveRegex.test(query);
};

const renderLearnMoreLink = () => {
  return (
    <CustomLink
      url="https://fleetdm.com/learn-more-about/vulnerability-processing"
      text="Learn more"
      newTab
    />
  );
};

const EmptyVulnerabilitiesTable = ({
  isPremiumTier,
  teamId,
  exploitedFilter,
  isSoftwareDisabled,
  searchQuery = "",
  knownVulnerability,
}: IEmptyVulnerabilitiesTableProps): JSX.Element => {
  const emptySoftware: IEmptyTableProps = {
    header: "No items match the current search criteria",
    info: "Expecting to see vulnerabilities? Check back later.",
  };

  if (searchQuery && !isValidCVEFormat(searchQuery)) {
    emptySoftware.header = "That vulnerability (CVE) is not valid";
    emptySoftware.info = (
      <>
        Try updating your search to use CVE format:
        <br />
        &quot;CVE-YYYY-&lt;4 or more digits&gt;&quot;
      </>
    );
  } else if ((!searchQuery || searchQuery === "") && !exploitedFilter) {
    emptySoftware.header = "No vulnerabilities detected";
  }

  if (knownVulnerability) {
    emptySoftware.header = `This is a known vulnerability (CVE), but it wasn't detected on any hosts${
      teamId !== undefined ? " in this team" : ""
    }.`;
    if (isPremiumTier && exploitedFilter) {
      emptySoftware.info =
        "If you're filtering by exploited CVEs, try removing the filter to expand your search.";
    }
    emptySoftware.additionalInfo = renderLearnMoreLink();
  } else if (knownVulnerability === false) {
    emptySoftware.header = "This is not a known CVE";
    emptySoftware.info =
      "None of Fleet's vulnerability sources are aware of this CVE.";
    emptySoftware.additionalInfo = renderLearnMoreLink();
  }

  if (isSoftwareDisabled) {
    emptySoftware.header = "Software inventory disabled";
    emptySoftware.info = (
      <>
        Users with the admin role can{" "}
        <CustomLink
          url="https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration"
          text="turn on software inventory"
          newTab
        />
        .
      </>
    );
  }

  return (
    <EmptyTable
      graphicName="empty-software"
      header={emptySoftware.header}
      info={emptySoftware.info}
      additionalInfo={emptySoftware.additionalInfo}
    />
  );
};

export default EmptyVulnerabilitiesTable;
