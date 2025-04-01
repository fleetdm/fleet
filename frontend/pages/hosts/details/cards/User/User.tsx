import React from "react";
import classnames from "classnames";

import { IHostEndUser } from "interfaces/host";
import { HostPlatform } from "interfaces/platform";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";

import UserValue from "./components/UserValue";
import {
  generateFullNameTipContent,
  generateGroupsTipContent,
} from "./helpers";
// import { generateEmailValue } from "./helpers";

const baseClass = "user-card";

interface IUserProps {
  platform: HostPlatform;
  endUsers: IHostEndUser[];
  className?: string;
  onAddEndUser: () => void;
}

const User = ({ endUsers, className, onAddEndUser }: IUserProps) => {
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
      <div className={`${baseClass}__header`}>
        <CardHeader header="User" />
        <Button
          className={`${baseClass}__add-user-btn`}
          variant="text-link"
          onClick={onAddEndUser}
        >
          + Add user
        </Button>
      </div>

      <div className={`${baseClass}__content`}>
        <DataSet
          title={
            <TooltipWrapper
              tipContent="Email collected from your IdP during automatic enrollment (ADE)."
              position="top"
              showArrow
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
              // values={[...testvalues]}
              values={[
                "thisisareallyreallyreallylongemailaddress@longdomain.com",
                ...testvalues,
              ]}
            />
          }
        />
        <DataSet
          title={
            <TooltipWrapper
              tipContent={generateFullNameTipContent(endUsers)}
              position="top"
              showArrow
            >
              Full name (IdP)
            </TooltipWrapper>
          }
          value={<UserValue values={["test name"]} />}
        />
        <DataSet
          title={
            <TooltipWrapper
              tipContent={generateGroupsTipContent(endUsers)}
              position="top"
              showArrow
            >
              Groups (IdP)
            </TooltipWrapper>
          }
          value={<UserValue values={["test group"]} />}
        />
        <DataSet
          title="Google Chrome profiles"
          value={<UserValue values={["test group"]} />}
        />
      </div>
    </Card>
  );
};

export default User;
