import React, { useContext, useEffect, useState } from "react";
import { formatDistanceToNow } from "date-fns";

import { IUser } from "interfaces/user";
import { IVersionData } from "interfaces/version";

import { AppContext } from "context/app";

import versionAPI from "services/entities/version";

import Avatar from "components/Avatar";
import Button from "components/buttons/Button";

import { generateRole, generateTeam, greyCell } from "utilities/helpers";

interface IUserSidePanelProps {
  currentUser: IUser;
  onChangePassword: () => void;
  onGetApiToken: () => void;
}

const baseClass = "user-side-panel";

const UserSidePanel = ({
  currentUser,
  onChangePassword,
  onGetApiToken,
}: IUserSidePanelProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
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

  const lastUpdatedAt =
    updatedAt &&
    formatDistanceToNow(new Date(updatedAt), {
      addSuffix: true,
    });

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
        <div className={`${baseClass}__more-info-detail`}>
          <p className={`${baseClass}__header`}>Teams</p>
          <p
            className={`${baseClass}__description ${baseClass}__teams ${greyCell(
              teamsText
            )}`}
          >
            {teamsText}
          </p>
        </div>
      )}
      <div className={`${baseClass}__more-info-detail`}>
        <p className={`${baseClass}__header`}>Role</p>
        <p
          className={`${baseClass}__description ${baseClass}__role ${greyCell(
            roleText
          )}`}
        >
          {roleText}
        </p>
      </div>
      <div className={`${baseClass}__more-info-detail`}>
        <p className={`${baseClass}__header`}>Password</p>
      </div>
      <Button
        onClick={onChangePassword}
        disabled={ssoEnabled}
        className={`${baseClass}__button`}
      >
        Change password
      </Button>
      <p className={`${baseClass}__last-updated`}>
        Last changed: {lastUpdatedAt}
      </p>
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

export default UserSidePanel;
