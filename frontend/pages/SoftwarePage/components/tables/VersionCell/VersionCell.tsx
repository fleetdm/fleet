import React from "react";
import { CellProps } from "react-table";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { formatSoftwareVersion, SoftwareSource } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IVersionEntry {
  version: string;
  release?: string;
}

interface IVersionCellProps<T extends IVersionEntry> {
  versions: T[] | null | undefined;
  /** Source of the software these versions belong to. */
  source: SoftwareSource;
}

const VersionCell = <T extends IVersionEntry>({
  versions,
  source,
}: IVersionCellProps<T>) => {
  if (!versions || versions.length === 0 || versions[0].version === "") {
    return <TextCell value={DEFAULT_EMPTY_CELL_VALUE} grey />;
  }

  const displayedVersions = versions.map((v) =>
    formatSoftwareVersion({ ...v, source })
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

/** react-table `Cell` for a column whose value is a row's version list. Version
 * entries carry no source of their own, so this reads the row's. */
export const VersionsColumnCell = <T extends { source: SoftwareSource }>({
  cell,
  row,
}: CellProps<T, IVersionEntry[] | null | undefined>) => (
  <VersionCell versions={cell.value} source={row.original.source} />
);

export default VersionCell;
