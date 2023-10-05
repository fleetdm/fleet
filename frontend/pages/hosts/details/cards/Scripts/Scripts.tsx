import React from "react";

import Card from "components/Card";

const baseClass = "host-scripts-section";

interface IScriptsProps {}

const Scripts = ({}: IScriptsProps) => {
  return (
    <Card className={baseClass} borderRadiusSize="large" includeShadow>
      test
    </Card>
  );
};

export default Scripts;
