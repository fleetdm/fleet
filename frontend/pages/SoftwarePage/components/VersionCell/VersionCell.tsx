import React from "react";
import { uniqueId } from "lodash";

import { ISoftwareTitleVersion } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ReactTooltip from "react-tooltip";

const baseClass = "version-cell";

const generateVersionText = (versions: ISoftwareTitleVersion[]) => {
  const text =
    versions.length !== 1 ? `${versions.length} versions` : versions[0].version;
  return <TextCell value={text} greyed={versions.length !== 1} />;
};

const generateVersionTooltip = (
  versions: ISoftwareTitleVersion[],
  tooltipId: string
) => {
  if (versions.length <= 1) {
    return null;
  }

  return (
    <ReactTooltip
      effect="solid"
      backgroundColor="#3e4771"
      id={tooltipId}
      data-html
    >
      <ul className={`${baseClass}__version-list`}>
        {versions.map((version) => (
          <li>&bull; {version.version}</li>
        ))}
      </ul>
    </ReactTooltip>
  );
};

interface IVersionCellProps {
  versions: ISoftwareTitleVersion[];
}

const VersionCell = ({ versions }: IVersionCellProps) => {
  const tooltipId = uniqueId();

  // only one version, no need for tooltip
  const cellText = generateVersionText(versions);
  if (versions.length <= 1) {
    return <>{cellText}</>;
  }

  const versionTooltip = generateVersionTooltip(versions, tooltipId);
  return (
    <>
      <div
        className={`${baseClass}__version-text-with-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        {cellText}
      </div>
      {versionTooltip}
    </>
  );
};

export default VersionCell;
