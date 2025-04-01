import React from "react";
import classnames from "classnames";

import { IHostEndUser } from "interfaces/host";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import { HostPlatform } from "interfaces/platform";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";

import UserValue from "./UserValue";
// import { generateEmailValue } from "./helpers";

const baseClass = "user-card";

interface IUserProps {
  platform: HostPlatform;
  endUsers: IHostEndUser[];
  className?: string;
}

const User = ({ endUsers, className }: IUserProps) => {
  const classNames = classnames(baseClass, className);

  const testvalues = [];
  for (let i = 0; i < 30; i++) {
    testvalues.push(`test-email-${i}`);
  }

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <CardHeader header="User" />
      <div className={`${baseClass}__content`}>
        <DataSet
          title={
            <TooltipWrapper
              tipContent="Email collected from your IdP during automatic enrollment (ADE)."
              position="top"
            >
              Email
            </TooltipWrapper>
          }
          // value={generateEmailValue(endUsers)}
          value={
            <UserValue
              // values={[
              //   "thisisareallyreallyreallylongemailaddress@longdomain.com",
              // ]}
              // values={["shortemail@test.com"]}
              // values={["shortemail@test.com", "shortemail2@test.com"]}
              // values={[
              //   "thisisareallyreallyreallylongemailaddress@longdomain.com",
              //   "shortemail@longdomain.com",
              //   "thisisareallyreallyreallylongemailaddress2@longdomain.com",
              //   "mediumsidedemail@mediumdomain.com",
              // ]}
              values={[...testvalues]}
            />
          }
        />
      </div>
    </Card>
  );
};

export default User;
