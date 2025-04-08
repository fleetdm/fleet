import React from "react";
import classnames from "classnames";
import { noop } from "lodash";

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
  generateOtherEmailsValue,
} from "./helpers";

const baseClass = "user-card";

interface IUserProps {
  platform: HostPlatform;
  endUsers: IHostEndUser[];
  enableAddEndUser: boolean;
  disableFullNameTooltip?: boolean;
  disableGroupsTooltip?: boolean;
  className?: string;
  onAddEndUser?: () => void;
}

const User = ({
  platform,
  endUsers,
  enableAddEndUser,
  disableFullNameTooltip = false,
  disableGroupsTooltip = false,
  className,
  onAddEndUser = noop,
}: IUserProps) => {
  const classNames = classnames(baseClass, className);

  const userNameDisplayValues = generateUsernameValues(endUsers);
  const chromeProfilesDisplayValues = generateChromeProfilesValue(endUsers);

  const endUser = endUsers[0];
  const showUsername = platform === "darwin";
  const showFullName = showUsername && userNameDisplayValues.length > 0;
  const showGroups = showUsername && userNameDisplayValues.length > 0;
  const showChromeProfiles = chromeProfilesDisplayValues.length > 0;
  const showOtherEmails = endUser.other_emails.length > 0;

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <div className={`${baseClass}__header`}>
        <CardHeader header="User" />
        {enableAddEndUser && (
          <Button
            className={`${baseClass}__add-user-btn`}
            variant="text-link"
            onClick={onAddEndUser}
          >
            + Add user
          </Button>
        )}
      </div>

      <div className={`${baseClass}__content`}>
        {showUsername && (
          <DataSet
            title={
              <TooltipWrapper tipContent="Username collected from your IdP during automatic enrollment (ADE).">
                Username (IdP)
              </TooltipWrapper>
            }
            value={<UserValue values={userNameDisplayValues} />}
          />
        )}

        {showFullName && (
          <DataSet
            title={
              disableFullNameTooltip ? (
                "Full name (IdP)"
              ) : (
                <TooltipWrapper
                  tipContent={generateFullNameTipContent(endUsers)}
                >
                  Full name (IdP)
                </TooltipWrapper>
              )
            }
            value={<UserValue values={generateFullNameValues(endUsers)} />}
          />
        )}
        {showGroups && (
          <DataSet
            title={
              disableGroupsTooltip && endUser.idp_info_updated_at !== null ? (
                "Groups (IdP)"
              ) : (
                <TooltipWrapper tipContent={generateGroupsTipContent(endUsers)}>
                  Groups (IdP)
                </TooltipWrapper>
              )
            }
            value={<UserValue values={generateGroupsValues(endUsers)} />}
          />
        )}
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
            value={<UserValue values={generateOtherEmailsValue(endUsers)} />}
          />
        )}
      </div>
    </Card>
  );
};

export default User;
