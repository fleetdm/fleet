import React from "react";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";

import TextCell from "components/TableContainer/DataTable/TextCell";
import { ISoftwareVulnerability } from "interfaces/software";

const NUM_VULNERABILITIES_IN_TOOLTIP = 3;

const baseClass = "vulnerabilities-cell";

const generateCell = (
  vulnerabilities: ISoftwareVulnerability[] | string[] | null
) => {
  if (vulnerabilities === null) {
    return <TextCell value="---" greyed />;
  }

  let text = "";
  let isGrayed = true;
  if (vulnerabilities.length === 0) {
    text = "---";
  } else if (vulnerabilities.length === 1) {
    isGrayed = false;
    text =
      typeof vulnerabilities[0] === "string"
        ? vulnerabilities[0]
        : vulnerabilities[0].cve;
  } else {
    text = `${vulnerabilities.length} vulnerabilities`;
  }

  return <TextCell value={text} greyed={isGrayed} />;
};

const getName = (vulnerabiltiy: ISoftwareVulnerability | string) => {
  return typeof vulnerabiltiy === "string" ? vulnerabiltiy : vulnerabiltiy.cve;
};

const condenseVulnerabilities = (
  vulnerabilities: ISoftwareVulnerability[] | string[]
) => {
  const condensed =
    (vulnerabilities?.length &&
      vulnerabilities
        .slice(-NUM_VULNERABILITIES_IN_TOOLTIP)
        .map(getName)
        .reverse()) ||
    [];

  return vulnerabilities.length > NUM_VULNERABILITIES_IN_TOOLTIP
    ? condensed.concat(
        `+${vulnerabilities.length - NUM_VULNERABILITIES_IN_TOOLTIP} more`
      )
    : condensed;
};

const generateTooltip = (
  vulnerabilities: ISoftwareVulnerability[] | string[],
  tooltipId: string
) => {
  if (vulnerabilities.length <= 1) {
    return null;
  }

  const condensedVulnerabilties = condenseVulnerabilities(vulnerabilities);

  return (
    <ReactTooltip
      effect="solid"
      backgroundColor="#3e4771"
      id={tooltipId}
      data-html
    >
      <ul className={`${baseClass}__vulnerability-list`}>
        {condensedVulnerabilties.map((vulnerability) => {
          return <li>{vulnerability}</li>;
        })}
      </ul>
    </ReactTooltip>
  );
};
interface IVulnerabilitiesCellProps {
  vulnerabilities: ISoftwareVulnerability[] | string[] | null;
}

const VulnerabilitiesCell = ({
  vulnerabilities,
}: IVulnerabilitiesCellProps) => {
  const tooltipId = uniqueId();

  // only one vulnerability, no need for tooltip
  const cell = generateCell(vulnerabilities);
  if (vulnerabilities === null || vulnerabilities.length <= 1) {
    return <>{cell}</>;
  }

  const vulnerabilityTooltip = generateTooltip(vulnerabilities, tooltipId);

  return (
    <>
      <div
        className={`${baseClass}__vulnerability-text-with-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        {cell}
      </div>
      {vulnerabilityTooltip}
    </>
  );
};

export default VulnerabilitiesCell;
