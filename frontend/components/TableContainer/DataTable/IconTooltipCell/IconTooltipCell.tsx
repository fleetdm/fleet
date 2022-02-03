import React from "react";
import { isEmpty } from "lodash";

import IconToolTip from "components/IconToolTip";

interface IIconTooltipCellProps<T> {
  value: string;
}

const IconTooltipCell = ({
  value,
}: IIconTooltipCellProps<any>): JSX.Element | null => {
  if (isEmpty(value)) {
    return null;
  }

  return <IconToolTip text={value} issue isHtml />;
};

export default IconTooltipCell;
