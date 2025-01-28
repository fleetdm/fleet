import React from "react";

import { generateResultsCountText } from "../utilities/TableContainerUtils";

interface ITableCountProps {
  name: string;
  count?: number;
}

const TableCount = ({ name, count }: ITableCountProps): JSX.Element => {
  return <span>{generateResultsCountText(name, count)}</span>;
};

export default TableCount;
