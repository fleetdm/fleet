import React from "react";

import Card from "components/Card";

const baseClass = "details-no-hosts";

interface IDetailsNoHosts {
  header: string;
  details: string;
}

const DetailsNoHosts = ({ header, details }: IDetailsNoHosts) => {
  return (
    <Card borderRadiusSize="xxlarge" includeShadow className={baseClass}>
      <h2>{header}</h2>
      <p>{details}</p>
    </Card>
  );
};

export default DetailsNoHosts;
