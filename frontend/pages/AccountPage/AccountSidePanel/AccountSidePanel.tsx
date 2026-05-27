import React, { useContext, useEffect, useState } from "react";

import { IUser } from "interfaces/user";
import { IVersionData } from "interfaces/version";

import { AppContext } from "context/app";

import versionAPI from "services/entities/version";

import Avatar from "components/Avatar";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import Radio from "components/forms/fields/Radio";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import { generateRole, generateTeam, readableDate } from "utilities/helpers";
import { getThemeMode, setThemeMode, ThemeMode } from "utilities/theme";

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
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() =>
    getThemeMode()
  );
  const [hasGravatarPhoto, setHasGravatarPhoto] = useState(false);

  useEffect(() => {
    const url = currentUser.gravatar_url;
    if (!url) {
      setHasGravatarPhoto(false);
      return;
    }
    // Strip any existing d= param so Gravatar's built-in fallback (identicon,
    // mystery-person, etc.) doesn't mask the "no real photo" case, then add
    // d=404 so a missing photo errors out instead.
    const stripped = url.replace(/([?&])d=[^&]*&?/g, "$1").replace(/[?&]$/, "");
    const testUrl = stripped.includes("?")
      ? `${stripped}&d=404`
      : `${stripped}?d=404`;
    const img = new window.Image();
    img.onload = () => setHasGravatarPhoto(true);
    img.onerror = () => setHasGravatarPhoto(false);
    img.src = testUrl;
  }, [currentUser.gravatar_url]);

  const onThemeSelect = (value: string) => {
    const mode = value as ThemeMode;
    setThemeModeState(mode);
    setThemeMode(mode);
  };

  useEffect(() => {
    const getVersionData = async () => {
      try {
        const data = await versionAPI.load();
        setVersionData(data);
      } catch (response) {
        console.error(response);
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
        <div className={`${baseClass}__change-avatar-circle`}>
          <Avatar user={currentUser} className={`${baseClass}__avatar`} />
          <a
            className={`${baseClass}__change-photo-link`}
            href="https://en.gravatar.com/emails/"
            target="_blank"
            rel="noopener noreferrer"
          >
            Change photo at{" "}
            <span className={`${baseClass}__change-photo-link-nowrap`}>
              Gravatar
              <Icon
                name="external-link"
                className={`${baseClass}__change-photo-icon`}
                color="static-white"
              />
            </span>
          </a>
        </div>
        <a
          className={`${baseClass}__change-photo-badge`}
          href="https://en.gravatar.com/emails/"
          target="_blank"
          rel="noopener noreferrer"
          aria-hidden="true"
          tabIndex={-1}
        >
          <Icon
            name={hasGravatarPhoto ? "pencil" : "plus"}
            color="core-fleet-black"
            size="small"
          />
        </a>
      </div>
      <div
        className={`${baseClass}__theme-picker`}
        role="radiogroup"
        aria-label="Theme"
      >
        <div className={`${baseClass}__theme-picker-label`}>Theme</div>
        <Radio
          id="theme-system"
          name="theme"
          value="system"
          label="System"
          checked={themeMode === "system"}
          onChange={onThemeSelect}
        />
        <Radio
          id="theme-light"
          name="theme"
          value="light"
          label="Light"
          checked={themeMode === "light"}
          onChange={onThemeSelect}
        />
        <Radio
          id="theme-dark"
          name="theme"
          value="dark"
          label="Dark"
          checked={themeMode === "dark"}
          onChange={onThemeSelect}
        />
      </div>
      {isPremiumTier && <DataSet title="Fleets" value={teamsText} />}
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
