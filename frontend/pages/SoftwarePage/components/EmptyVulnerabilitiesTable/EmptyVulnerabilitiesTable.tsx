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

export const isValidCVEFormat = (query: string): boolean => {
  if (query.length < 9) {
    return false;
  }

  const cveRegex = /^(CVE-)?\d{4}-\d{4,5}$/i;
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
  const emptyVulns: IEmptyTableProps = {
    graphicName: "empty-search-question",
    header: "No items match the current search criteria",
    info: "Expecting to see vulnerabilities? Check back later.",
  };

  if (searchQuery && !isValidCVEFormat(searchQuery)) {
    emptyVulns.graphicName = "empty-search-exclamation";
    emptyVulns.header = "That vulnerability (CVE) is not valid";
    emptyVulns.info = (
      <>
        Try updating your search to use CVE format:
        <br />
        &quot;CVE-YYYY-&lt;4 or more digits&gt;&quot;
      </>
    );
  } else if (!searchQuery && !exploitedFilter) {
    emptyVulns.header = "No vulnerabilities detected";
  }

  if (knownVulnerability) {
    emptyVulns.graphicName = "empty-search-check";
    emptyVulns.header = `This is a known vulnerability (CVE), but it wasn't detected on any hosts${
      teamId !== undefined ? " in this team" : ""
    }.`;
    if (isPremiumTier && exploitedFilter) {
      emptyVulns.info =
        "Try removing the exploited vulnerabilities filter to expand your search.";
    }
    emptyVulns.additionalInfo = renderLearnMoreLink();
  } else if (knownVulnerability === false) {
    emptyVulns.graphicName = "empty-search-question";
    emptyVulns.header = "This is not a known CVE";
    emptyVulns.info =
      "None of Fleet's vulnerability sources are aware of this CVE.";
    emptyVulns.additionalInfo = renderLearnMoreLink();
  }

  if (isSoftwareDisabled) {
    emptyVulns.graphicName = "empty-search-question";
    emptyVulns.header = "Software inventory disabled";
    emptyVulns.info = (
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
      graphicName={emptyVulns.graphicName}
      header={emptyVulns.header}
      info={emptyVulns.info}
      additionalInfo={emptyVulns.additionalInfo}
    />
  );
};

export default EmptyVulnerabilitiesTable;
