import React from "react";

import { IFormattedDataPoint } from "./types";

const baseClass = "checkerboard-viz";

interface ICheckerboardVizProps {
  data: IFormattedDataPoint[];
  selectedDays: number;
  isPercentage: boolean;
}

const CheckerboardViz = ({
  data,
  selectedDays,
  isPercentage,
}: ICheckerboardVizProps): JSX.Element => {
  // TODO: implement checkerboard/heat map visualization
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__placeholder`}>
        Checkerboard visualization coming soon ({data.length} data points)
      </div>
    </div>
  );
};

export default CheckerboardViz;
