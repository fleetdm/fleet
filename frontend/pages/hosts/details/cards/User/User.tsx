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
  generateChromeProfilesValue,
  generateUsernameValues,
  generateFullNameTipContent,
  generateFullNameValues,
  generateGroupsTipContent,
  generateGroupsValues,
} from "./helpers";

const baseClass = "user-card";

interface IUserProps {
  platform: HostPlatform;
  endUsers: IHostEndUser[];
  className?: string;
  onAddEndUser: () => void;
}

const User = ({ platform, endUsers, className, onAddEndUser }: IUserProps) => {
  const classNames = classnames(baseClass, className);

  // TODO: show and hide data sets
  const endUser = endUsers[0];
  const showUsername = platform === "darwin";

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
            <TooltipWrapper tipContent="Username collected from your IdP during automatic enrollment (ADE).">
              Username (IdP)
            </TooltipWrapper>
          }
          value={<UserValue values={generateUsernameValues(endUsers)} />}
        />
        <DataSet
          title={
            <TooltipWrapper tipContent={generateFullNameTipContent(endUsers)}>
              Full name (IdP)
            </TooltipWrapper>
          }
          value={<UserValue values={generateFullNameValues(endUsers)} />}
        />
        <DataSet
          title={
            endUsers[0].idp_info_updated_at === null ? (
              <TooltipWrapper tipContent={generateGroupsTipContent(endUsers)}>
                Groups (IdP)
              </TooltipWrapper>
            ) : (
              "Groups (IdP)"
            )
          }
          value={<UserValue values={generateGroupsValues(endUsers)} />}
        />
        <DataSet
          title="Google Chrome profiles"
          value={<UserValue values={generateChromeProfilesValue(endUsers)} />}
        />
      </div>
    </Card>
  );
};

export default User;
