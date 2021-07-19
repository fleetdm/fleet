import React from "react";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

interface IIconTooltipCellProps<T> {
  value: string;
}

const IconTooltipCell = (
  props: IIconTooltipCellProps<any>
): JSX.Element | null => {
  const { value } = props;

  // The value passed in must be a valid FleetIcon name
  return <FleetIcon name={value} />;
};

export default IconTooltipCell;
