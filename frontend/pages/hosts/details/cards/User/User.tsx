import React from "react";
import classnames from "classnames";

import { IHostEndUser } from "interfaces/host";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import UserValue from "./components/UserValue";
import {
  generateChromeProfilesValues,
  generateUsernameValues,
  generateFullNameTipContent,
  generateFullNameValues,
  generateGroupsTipContent,
  generateGroupsValues,
  generateOtherEmailsValues,
} from "./helpers";

const baseClass = "user-card";

interface IUserProps {
  /** There will be at most 1 end user */
  endUsers: IHostEndUser[];
  canWriteEndUser?: boolean;
  disableFullNameTooltip?: boolean;
  disableGroupsTooltip?: boolean;
  className?: string;
  onClickUpdateUser?: (
    e:
      | React.MouseEvent<HTMLButtonElement>
      | React.KeyboardEvent<HTMLButtonElement>
  ) => void;
}

const User = ({
  endUsers,
  canWriteEndUser = false,
  disableFullNameTooltip = false,
  disableGroupsTooltip = false,
  className,
  onClickUpdateUser,
}: IUserProps) => {
  const classNames = classnames(baseClass, className);

  // though this code implies otherwise, there will be at most 1 end user
  const userNameDisplayValues = generateUsernameValues(endUsers);
  const chromeProfilesDisplayValues = generateChromeProfilesValues(endUsers);
  const otherEmailsDisplayValues = generateOtherEmailsValues(endUsers);

  const [writeButtonText, writeButtonIcon] = userNameDisplayValues.length
    ? ["Edit user", "pencil" as const]
    : ["Add user", "plus" as const];

  const endUser = endUsers[0];
  const showChromeProfiles = chromeProfilesDisplayValues.length > 0;
  const showOtherEmails = otherEmailsDisplayValues.length > 0;
  const userDepartment = [];
  if (endUser?.idp_department) {
    userDepartment.push(endUser.idp_department);
  }
  const groupsTipContent = generateGroupsTipContent(endUsers);

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
    >
      <div className={`${baseClass}__header`}>
        <CardHeader header="User" />
        {canWriteEndUser && (
          <Button
            className={`${baseClass}__add-user-btn`}
            variant="inverse"
            onClick={onClickUpdateUser}
            size="small"
          >
            <Icon name={writeButtonIcon} />
            {writeButtonText}
          </Button>
        )}
      </div>

      <div className={`${baseClass}__content`}>
        <DataSet
          title="Username (IdP)"
          value={<UserValue values={userNameDisplayValues} />}
        />

        <DataSet
          title={
            disableFullNameTooltip ? (
              "Full name (IdP)"
            ) : (
              <TooltipWrapper tipContent={generateFullNameTipContent(endUsers)}>
                Full name (IdP)
              </TooltipWrapper>
            )
          }
          value={<UserValue values={generateFullNameValues(endUsers)} />}
        />
        <DataSet
          title={
            disableGroupsTooltip || !groupsTipContent ? (
              "Groups (IdP)"
            ) : (
              <TooltipWrapper tipContent={groupsTipContent}>
                <>Groups (IdP)</>
              </TooltipWrapper>
            )
          }
          value={<UserValue values={generateGroupsValues(endUsers)} />}
        />
        <DataSet
          title={
            <TooltipWrapper tipContent='This is the "department" collected from your IdP.'>
              Department (IdP)
            </TooltipWrapper>
          }
          value={<UserValue values={userDepartment} />}
        />
        {showChromeProfiles && (
          <DataSet
            title="Google Chrome profiles"
            value={<UserValue values={chromeProfilesDisplayValues} />}
          />
        )}
        {showOtherEmails && (
          <DataSet
            title={
              <TooltipWrapper tipContent="Custom email added to the host via custom human-device mapping API.">
                Other emails
              </TooltipWrapper>
            }
            value={<UserValue values={generateOtherEmailsValues(endUsers)} />}
          />
        )}
      </div>
    </Card>
  );
};

export default User;
