import React from "react";
import classnames from "classnames";

import { IHostEndUser } from "interfaces/host";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import { HostPlatform } from "interfaces/platform";

const baseClass = "user-card";

interface IUserProps {
  platform: HostPlatform;
  endUsers: IHostEndUser[];
  className?: string;
}

const User = ({ endUsers, className }: IUserProps) => {
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
