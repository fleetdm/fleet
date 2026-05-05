import React from "react";
import { PlacesType } from "react-tooltip-5";

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

const getName = (vulnerability: ISoftwareVulnerability | string) => {
  return typeof vulnerability === "string" ? vulnerability : vulnerability.cve;
};

const condenseVulnerabilities = (
  vulnerabilities: ISoftwareVulnerability[] | string[],
  totalCount?: number
) => {
  const condensed =
    vulnerabilities.length > 0
      ? vulnerabilities
          .slice(-NUM_VULNERABILITIES_IN_TOOLTIP)
          .map(getName)
          .reverse()
      : [];

  const count = totalCount ?? vulnerabilities.length;

  return count > NUM_VULNERABILITIES_IN_TOOLTIP
    ? condensed.concat(`+${count - NUM_VULNERABILITIES_IN_TOOLTIP} more`)
    : condensed;
};

const generateTooltip = (
  vulnerabilities: ISoftwareVulnerability[] | string[],
  totalCount?: number
) => {
  const count = totalCount ?? vulnerabilities.length;

  if (count <= 1) {
    return null;
  }

  const condensedVulnerabilities = condenseVulnerabilities(
    vulnerabilities,
    totalCount
  );

  return (
    <ul className={`${baseClass}__vulnerability-list`}>
      {condensedVulnerabilities.map((vulnerability) => (
        <li key={vulnerability}>{vulnerability}</li>
      ))}
    </ul>
  );
};

interface IVulnerabilitiesCellProps {
  vulnerabilities: ISoftwareVulnerability[] | string[] | null;
  vulnerabilitiesCount?: number;
  tooltipPosition?: PlacesType;
}

const VulnerabilitiesCell = ({
  vulnerabilities,
  vulnerabilitiesCount,
  tooltipPosition = "top",
}: IVulnerabilitiesCellProps) => {
  const cell = generateCell(vulnerabilities, vulnerabilitiesCount);
  const count = vulnerabilitiesCount ?? vulnerabilities?.length ?? 0;

  if (vulnerabilities === null || count <= 1) {
    return <>{cell}</>;
  }

  const vulnerabilityTooltip = generateTooltip(
    vulnerabilities,
    vulnerabilitiesCount
  );

  return (
    <TooltipWrapper
      tipContent={vulnerabilityTooltip}
      position={tooltipPosition}
      underline={false}
      showArrow
    >
      <div className={`${baseClass}__vulnerability-text-with-tooltip`}>
        {cell}
      </div>
    </TooltipWrapper>
  );
};

export default VulnerabilitiesCell;
