import React from "react";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IEmptyCellWithTooltipProps {
  /** Why the cell is empty. Renders a plain grey "---" when omitted. */
  tipContent?: React.ReactNode;
}

const EmptyCellWithTooltip = ({ tipContent }: IEmptyCellWithTooltipProps) => {
  const cell = <TextCell value={DEFAULT_EMPTY_CELL_VALUE} grey />;
  if (!tipContent) {
    return cell;
  }
  return (
    <TooltipWrapper
      tipContent={tipContent}
      position="top"
      showArrow
      underline={false}
      fixedPositionStrategy
    >
      {cell}
    </TooltipWrapper>
  );
};

export default EmptyCellWithTooltip;
