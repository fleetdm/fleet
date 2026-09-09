import React from "react";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import { formatSoftwareVersion } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IVersionCellProps<T extends { version: string; release?: string }> {
  versions: T[] | null | undefined;
  /** Source of the software these versions belong to. */
  source: string;
}

const VersionCell = <T extends { version: string; release?: string }>({
  versions,
  source,
}: IVersionCellProps<T>) => {
  if (!versions || versions.length === 0 || versions[0].version === "") {
    return <TextCell value={DEFAULT_EMPTY_CELL_VALUE} grey />;
  }

  const displayedVersions = versions.map(({ version, release }) =>
    formatSoftwareVersion({ version, release, source })
  );

  if (displayedVersions.length === 1) {
    return <TooltipTruncatedTextCell value={displayedVersions[0]} />;
  }

  // Multiple versions: show count, tooltip with versions list
  return (
    <TooltipWrapper
      tipContent={<>{displayedVersions.join(", ")}</>}
      tipOffset={14}
      position="top"
      showArrow
      underline={false}
      fixedPositionStrategy
    >
      <TextCell value={`${displayedVersions.length} versions`} italic />
    </TooltipWrapper>
  );
};

export default VersionCell;
