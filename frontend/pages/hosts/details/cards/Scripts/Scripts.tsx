import React from "react";

import Card from "components/Card";
import TableContainer from "components/TableContainer";

const baseClass = "host-scripts-section";

interface IScriptsProps {}

const Scripts = ({}: IScriptsProps) => {
  return (
    <Card className={baseClass} borderRadiusSize="large" includeShadow>
      <h2>Scripts</h2>
      <TableContainer />
    </Card>
  );
};

export default Scripts;
