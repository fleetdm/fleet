import React from "react";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";

import TextCell from "components/TableContainer/DataTable/TextCell";
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

const generateTooltip = (
  vulnerabilities: ISoftwareVulnerability[] | string[],
  tooltipId: string,
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
    <ReactTooltip
      effect="solid"
      backgroundColor="#3e4771"
      id={tooltipId}
      data-html
    >
      <ul className={`${baseClass}__vulnerability-list`}>
        {condensedVulnerabilities.map((vulnerability) => {
          const key =
            typeof vulnerability === "string" ? vulnerability : uniqueId();
          return <li key={key}>{vulnerability}</li>;
        })}
      </ul>
    </ReactTooltip>
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
  const tooltipId = uniqueId();

  // only one vulnerability, no need for tooltip
  const cell = generateCell(vulnerabilities, vulnerabilitiesCount);
  const count = vulnerabilitiesCount ?? vulnerabilities?.length ?? 0;
  if (vulnerabilities === null || count <= 1) {
    return <>{cell}</>;
  }

  const vulnerabilityTooltip = generateTooltip(
    vulnerabilities,
    tooltipId,
    vulnerabilitiesCount
  );

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
