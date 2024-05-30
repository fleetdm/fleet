import React from "react";
import { uniqueId } from "lodash";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ReactTooltip from "react-tooltip";

const baseClass = "version-cell";

const generateText = <T extends { version: string }>(versions: T[] | null) => {
  if (!versions) {
    return <TextCell value="---" greyed />;
  }
  const text =
    versions.length !== 1 ? `${versions.length} versions` : versions[0].version;
  return <TextCell value={text} greyed={versions.length !== 1} />;
};

const generateTooltip = <T extends { version: string }>(
  versions: T[],
  tooltipId: string
) => {
  if (!versions) {
    return null;
  }

  const versionNames = versions.map((version) => version.version);

  return (
    <ReactTooltip
      effect="solid"
      backgroundColor="#3e4771"
      id={tooltipId}
      data-html
    >
      <p className={`${baseClass}__versions`}>{versionNames.join(", ")}</p>
    </ReactTooltip>
  );
};

interface IVersionCellProps<T extends { version: string }> {
  versions: T[] | null;
}

const VersionCell = <T extends { version: string }>({
  versions,
}: IVersionCellProps<T>) => {
  // only one version, no need for tooltip
  const cellText = generateText(versions);
  if (!versions || versions.length <= 1) {
    return <>{cellText}</>;
  }

  const tooltipId = uniqueId();

  const versionTooltip = generateTooltip(versions, tooltipId);
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
