import React from "react";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

interface IIconTooltipCellProps<T> {
  value: string;
}

const IconTooltipCell = ({
  value,
}: IIconTooltipCellProps<any>): JSX.Element | null => {
  // The value passed in must be a valid FleetIcon name
  return <FleetIcon name={value} />;
};

export default IconTooltipCell;
