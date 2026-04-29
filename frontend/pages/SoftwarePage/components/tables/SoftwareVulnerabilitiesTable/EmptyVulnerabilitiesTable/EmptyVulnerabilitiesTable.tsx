import React from "react";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import { IEmptyStateProps } from "interfaces/empty_state";
import { IVulnerabilitiesEmptyStateReason } from "services/entities/vulnerabilities";

export interface IEmptyVulnerabilitiesTableProps {
  isPremiumTier?: boolean;
  teamId?: number;
  exploitedFilter?: boolean;
  isSoftwareDisabled?: boolean;
  emptyStateReason?: IVulnerabilitiesEmptyStateReason;
}

const LearnMoreLink = () => (
  <CustomLink
    url="https://fleetdm.com/learn-more-about/vulnerability-processing"
    text="Learn more"
    newTab
  />
);

const emptyStateDetails: Record<
  IVulnerabilitiesEmptyStateReason,
  Partial<IEmptyStateProps>
> = {
  "no-vulns-detected": {
    header: "No vulnerabilities detected",
    info: "Expecting to see vulnerabilities? Check back later.",
  },
  "no-matching-items": {
    header: "No items match the current search criteria",
    info: "Expecting to see vulnerabilities? Check back later.",
  },
  "invalid-cve": {
    header: "That vulnerability (CVE) is not valid",
    info:
      'Try updating your search to use CVE format: "CVE-YYYY-<4 or more digits>"',
  },
  "unknown-cve": {
    header: "This is not a known CVE",
    info: "None of Fleet's vulnerability sources are aware of this CVE.",
    additionalInfo: <LearnMoreLink />,
  },
  "known-vuln": {
    header:
      "This is a known vulnerability (CVE), but it wasn't detected on any hosts",
    additionalInfo: <LearnMoreLink />,
  },
};

const EmptyVulnerabilitiesTable: React.FC<IEmptyVulnerabilitiesTableProps> = ({
  isPremiumTier,
  teamId,
  exploitedFilter,
  isSoftwareDisabled,
  emptyStateReason,
}) => {
  if (isSoftwareDisabled) {
    return (
      <EmptyState
        header="Software inventory disabled"
        info={
          <>
            Users with the admin role can{" "}
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration"
              text="turn on software inventory"
              newTab
            />
            .
          </>
        }
      />
    );
  }

  const defaultEmptyState: IEmptyStateProps = {
    header: "No items match the current search criteria",
    info: "Expecting to see vulnerabilities? Check back later.",
  };

  const emptyState = emptyStateReason
    ? { ...defaultEmptyState, ...emptyStateDetails[emptyStateReason] }
    : defaultEmptyState;

  if (emptyStateReason === "known-vuln" && teamId !== undefined) {
    emptyState.header += " in this fleet";
  }

  if (
    isPremiumTier &&
    exploitedFilter &&
    emptyStateReason !== "unknown-cve" &&
    emptyStateReason !== "invalid-cve"
  ) {
    emptyState.info =
      "Try removing the exploited vulnerabilities filter to expand your search.";
  }

  return <EmptyState {...emptyState} />;
};

export default EmptyVulnerabilitiesTable;
