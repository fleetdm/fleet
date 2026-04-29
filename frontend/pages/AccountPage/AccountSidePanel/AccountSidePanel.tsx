import React, { useContext, useEffect, useState } from "react";

import { IUser } from "interfaces/user";
import { IVersionData } from "interfaces/version";

import { AppContext } from "context/app";

import versionAPI from "services/entities/version";

import Avatar from "components/Avatar";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import {
  generateRole,
  generateTeam,
  greyCell,
  readableDate,
} from "utilities/helpers";
import { isDarkMode, toggleDarkMode } from "utilities/theme";

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
  const [darkMode, setDarkMode] = useState(() => isDarkMode());

  useEffect(() => {
    const onThemeChange = (e: Event) => {
      setDarkMode((e as CustomEvent).detail.dark);
    };
    window.addEventListener("fleet-theme-change", onThemeChange);
    return () =>
      window.removeEventListener("fleet-theme-change", onThemeChange);
  }, []);

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
        <CustomLink
          url="https://en.gravatar.com/emails/"
          text="Change photo at Gravatar"
          newTab
        />
      </div>
      <div className={`${baseClass}__theme-toggle`}>
        <svg
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
          className={`${baseClass}__theme-icon`}
        >
          <circle
            cx="8"
            cy="8"
            r="3.5"
            stroke="currentColor"
            strokeWidth="1.5"
          />
          <path
            d="M8 1v2M8 13v2M1 8h2M13 8h2M3.05 3.05l1.41 1.41M11.54 11.54l1.41 1.41M3.05 12.95l1.41-1.41M11.54 4.46l1.41-1.41"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>
        <button
          type="button"
          role="switch"
          aria-checked={darkMode}
          aria-label="Toggle dark mode"
          className={`button button--unstyled ${baseClass}__toggle ${
            darkMode ? `${baseClass}__toggle--active` : ""
          }`}
          onClick={() => setDarkMode(toggleDarkMode())}
        >
          <div
            className={`${baseClass}__toggle-dot ${
              darkMode ? `${baseClass}__toggle-dot--active` : ""
            }`}
          />
        </button>
        <svg
          width="14"
          height="14"
          viewBox="0 0 16 16"
          fill="none"
          className={`${baseClass}__theme-icon`}
        >
          <path
            d="M14.3 10.7A7 7 0 0 1 5.3 1.7 7 7 0 1 0 14.3 10.7Z"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinejoin="round"
          />
        </svg>
      </div>
      {isPremiumTier && (
        <DataSet
          title="Fleets"
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
        <CustomLink
          url="https://fleetdm.com/legal/privacy"
          text="Privacy policy"
          newTab
        />
      </span>
    </div>
  );
};

export default AccountSidePanel;
