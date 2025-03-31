import React from "react";
import classnames from "classnames";

import Card from "components/Card";
import CardHeader from "components/CardHeader";

const baseClass = "user-card";

interface IUserProps {
  className?: string;
}

const User = ({ className }: IUserProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <CardHeader header="User" />
    </Card>
  );
};

export default User;
