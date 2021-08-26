import React from "react";
import { isEmpty } from "lodash";

import IconToolTip from "components/IconToolTip";

interface IIconTooltipCellProps<T> {
  value: string;
}

const IconTooltipCell = (
  props: IIconTooltipCellProps<any>
): JSX.Element | null => {
  const { value } = props;

  if (isEmpty(value)) {
    return null;
  }

  return <IconToolTip text={value} issue isHtml />;
};

export default IconTooltipCell;
