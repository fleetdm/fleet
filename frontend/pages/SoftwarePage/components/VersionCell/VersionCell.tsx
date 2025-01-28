import React from "react";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

const generateText = <T extends { version: string }>(versions: T[] | null) => {
  if (!versions) {
    return <TextCell value="---" grey />;
  }
  const text =
    versions.length !== 1 ? `${versions.length} versions` : versions[0].version;
  return <TextCell value={text} italic={versions.length !== 1} />;
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
      tipContent={<>{versions.map((version) => version.version).join(", ")}</>}
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
