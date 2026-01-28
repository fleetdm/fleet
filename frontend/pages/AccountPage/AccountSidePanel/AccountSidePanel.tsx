import React, { useContext, useEffect, useState } from "react";

import { IUser } from "interfaces/user";
import { IVersionData } from "interfaces/version";

import { AppContext } from "context/app";

import versionAPI from "services/entities/version";

import Avatar from "components/Avatar";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import {
  generateRole,
  generateTeam,
  greyCell,
  readableDate,
} from "utilities/helpers";

interface IAccountSidePanelProps {
  currentUser: IUser;
  onChangePassword: () => void;
  onGetApiToken: () => void;
}

const baseClass = "account-side-panel";

const AccountSidePanel = ({
  currentUser,
  onChangePassword,
  onGetApiToken,
}: IAccountSidePanelProps): JSX.Element => {
  const { isPremiumTier, config } = useContext(AppContext);
  const [versionData, setVersionData] = useState<IVersionData>();

  useEffect(() => {
    const getVersionData = async () => {
      try {
        const data = await versionAPI.load();
        setVersionData(data);
      } catch (response) {
        console.error(response);
        return false;
      }
    };

    getVersionData();
  }, []);

  const {
    global_role: globalRole,
    updated_at: updatedAt,
    sso_enabled: ssoEnabled,
    teams,
  } = currentUser;

  const roleText = generateRole(teams, globalRole);
  const teamsText = generateTeam(teams, globalRole);

  const lastUpdatedAt = updatedAt && (
    <HumanTimeDiffWithDateTip timeString={updatedAt} />
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__change-avatar`}>
        <Avatar user={currentUser} className={`${baseClass}__avatar`} />
        <a
          href="https://en.gravatar.com/emails/"
          target="_blank"
          rel="noopener noreferrer"
        >
          Change photo at Gravatar
        </a>
      </div>
      {isPremiumTier && (
        <DataSet
          title="Teams"
          value={
            <span
              className={`${
                greyCell(teamsText) ? `${baseClass}__grey-text` : ""
              }`}
            >
              {teamsText}
            </span>
          }
        />
      )}
      <DataSet title="Role" value={roleText} />
      {isPremiumTier && config && (
        <DataSet
          title="License expiration date"
          value={readableDate(config.license.expiration)}
        />
      )}
      <DataSet
        title="Password"
        value={
          <div className={`${baseClass}__password-info`}>
            <Button
              onClick={onChangePassword}
              disabled={ssoEnabled}
              className={`${baseClass}__button`}
            >
              Change password
            </Button>
            <div className={`${baseClass}__last-updated`}>
              Last changed: {lastUpdatedAt}
            </div>
          </div>
        }
      />
      <Button onClick={onGetApiToken} className={`${baseClass}__button`}>
        Get API token
      </Button>
      <span
        className={`${baseClass}__version`}
      >{`Fleet ${versionData?.version} • Go ${versionData?.go_version}`}</span>
      <span className={`${baseClass}__privacy-policy`}>
        <a
          href="https://fleetdm.com/legal/privacy"
          target="_blank"
          rel="noopener noreferrer"
        >
          Privacy policy
        </a>
      </span>
    </div>
  );
};

export default AccountSidePanel;
