import React from "react";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "version-cell";

const generateText = <T extends { version: string }>(versions: T[] | null) => {
  if (!versions) {
    return <TextCell value="---" greyed />;
  }
  const text =
    versions.length !== 1 ? `${versions.length} versions` : versions[0].version;
  return <TextCell value={text} greyed={versions.length !== 1} />;
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

  return (
    <TooltipWrapper
      tipContent={
        <p className={`${baseClass}__versions`}>
          {versions.map((version) => version.version).join(", ")}
        </p>
      }
      tipOffset={14}
      position="top"
      showArrow
      underline={false}
    >
      {cellText}
    </TooltipWrapper>
  );
};

export default VersionCell;
