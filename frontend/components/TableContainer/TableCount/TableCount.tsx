import React from "react";

import { generateResultsCountText } from "../utilities/TableContainerUtils";

const baseClass = "table-count";

interface ITableCountProps {
  name: string;
  count?: number;
}

const TableCount = ({ name, count }: ITableCountProps): JSX.Element => {
  return (
    <span className={baseClass}>{generateResultsCountText(name, count)}</span>
  );
};

export default TableCount;
