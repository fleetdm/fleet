import React from "react";
import { isEmpty } from "lodash";

import IconToolTip from "components/IconToolTip";

interface IIconTooltipCellProps<T> {
  value: string;
}

const IconTooltipCell = ({
  value,
}: IIconTooltipCellProps<any>): JSX.Element => {
  if (isEmpty(value)) {
    return <></>;
  }

  return <IconToolTip text={value} issue isHtml />;
};

export default IconTooltipCell;
