import React from "react";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IVersionCellProps<T extends { version: string }> {
  versions: T[] | null;
}

const VersionCell = <T extends { version: string }>({
  versions,
}: IVersionCellProps<T>) => {
  if (!versions || versions.length === 0) {
    return <TextCell value={DEFAULT_EMPTY_CELL_VALUE} grey />;
  }

  if (versions.length === 1) {
    return <TooltipTruncatedTextCell value={versions[0].version} />;
  }

  // Multiple versions: show count, tooltip with versions list
  return (
    <TooltipWrapper
      tipContent={<>{versions.map((version) => version.version).join(", ")}</>}
      tipOffset={14}
      position="top"
      showArrow
      underline={false}
    >
      <TextCell value={`${versions.length} versions`} italic />
    </TooltipWrapper>
  );
};

export default VersionCell;
