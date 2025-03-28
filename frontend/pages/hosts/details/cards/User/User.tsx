import Card from "components/Card";
import CardHeader from "components/CardHeader";
import React from "react";

const baseClass = "user-card";

interface IUserProps {}

const User = ({}: IUserProps) => {
  return (
    <Card
      className={baseClass}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <CardHeader header="User" />
    </Card>
  );
};

export default User;
