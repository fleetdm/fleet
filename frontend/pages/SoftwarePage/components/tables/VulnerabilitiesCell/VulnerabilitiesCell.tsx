import React from "react";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { ISoftwareVulnerability } from "interfaces/software";

const NUM_VULNERABILITIES_IN_TOOLTIP = 3;

const baseClass = "vulnerabilities-cell";

const generateCell = (
  vulnerabilities: ISoftwareVulnerability[] | string[] | null,
  vulnerabilitiesCount?: number
) => {
  if (vulnerabilities === null || vulnerabilities.length === 0) {
    return <TextCell value="---" grey />;
  }

  const totalCount = vulnerabilitiesCount ?? vulnerabilities.length;

  let text = "";
  let italicize = true;
  if (totalCount === 1) {
    italicize = false;
    text =
      typeof vulnerabilities[0] === "string"
        ? vulnerabilities[0]
        : vulnerabilities[0].cve;
  } else {
    text = `${totalCount} vulnerabilities`;
  }

  return <TextCell value={text} italic={italicize} />;
};

const getName = (vulnerabiltiy: ISoftwareVulnerability | string) => {
  return typeof vulnerabiltiy === "string" ? vulnerabiltiy : vulnerabiltiy.cve;
};

const condenseVulnerabilities = (
  vulnerabilities: ISoftwareVulnerability[] | string[],
  totalCount?: number
) => {
  const condensed =
    (vulnerabilities?.length &&
      vulnerabilities
        .slice(-NUM_VULNERABILITIES_IN_TOOLTIP)
        .map(getName)
        .reverse()) ||
    [];

  const count = totalCount ?? vulnerabilities.length;

  return count > NUM_VULNERABILITIES_IN_TOOLTIP
    ? condensed.concat(`+${count - NUM_VULNERABILITIES_IN_TOOLTIP} more`)
    : condensed;
};

const generateTooltipContent = (
  vulnerabilities: ISoftwareVulnerability[] | string[],
  totalCount?: number
) => {
  const condensedVulnerabilities = condenseVulnerabilities(
    vulnerabilities,
    totalCount
  );

  return (
    <ul className={`${baseClass}__vulnerability-list`}>
      {condensedVulnerabilities.map((vulnerability, index) => (
        // allow index in key as vulnerablity is not certain to be unique
        // eslint-disable-next-line react/no-array-index-key
        <li key={`vulnerability-${index}`}>{vulnerability}</li>
      ))}
    </ul>
  );
};
interface IVulnerabilitiesCellProps {
  vulnerabilities: ISoftwareVulnerability[] | string[] | null;
  vulnerabilitiesCount?: number;
}

const VulnerabilitiesCell = ({
  vulnerabilities,
  vulnerabilitiesCount,
}: IVulnerabilitiesCellProps) => {
  // only one vulnerability, no need for tooltip
  const cell = generateCell(vulnerabilities, vulnerabilitiesCount);
  const count = vulnerabilitiesCount ?? vulnerabilities?.length ?? 0;
  if (vulnerabilities === null || count <= 1) {
    return <>{cell}</>;
  }

  return (
    <TooltipWrapper
      showArrow
      tipContent={generateTooltipContent(vulnerabilities, vulnerabilitiesCount)}
      underline={false}
      className={`${baseClass}__vulnerability-text-with-tooltip`}
    >
      {cell}
    </TooltipWrapper>
  );
};

export default VulnerabilitiesCell;
