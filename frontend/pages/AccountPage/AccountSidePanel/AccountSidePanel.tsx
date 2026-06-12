import React, { useContext, useEffect, useState } from "react";

import { IUser } from "interfaces/user";
import { IVersionResponse } from "interfaces/version";

import { AppContext } from "context/app";

import versionAPI from "services/entities/version";

import Avatar from "components/Avatar";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";

import {
  generateRole,
  generateTeam,
  readableDate,
  tooltipTextWithLineBreaks,
} from "utilities/helpers";
import stringUtils from "utilities/strings";
import TooltipWrapper from "components/TooltipWrapper";
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
  const [versionData, setVersionData] = useState<IVersionResponse>();
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() =>
    getThemeMode()
  );

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

  const teamNames = (() => {
    const names = teams.map((t) => t.name);
    return globalRole !== null && teams.length > 0
      ? ["Global", ...names]
      : names;
  })();

  const roleGroups = (() => {
    const groups: { role: string; names: string[] }[] = [];
    if (globalRole !== null && teams.length > 0) {
      groups.push({
        role: stringUtils.capitalizeRole(globalRole),
        names: ["Global"],
      });
    }
    teams.forEach((team) => {
      const role = stringUtils.capitalizeRole(team.role || "Unassigned");
      const existing = groups.find((g) => g.role === role);
      if (existing) {
        existing.names.push(team.name);
      } else {
        groups.push({ role, names: [team.name] });
      }
    });
    return groups;
  })();

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
      {isPremiumTier && (
        <DataSet
          title="Fleets"
          value={
            teamNames.length > 1 ? (
              <TooltipWrapper
                tipContent={tooltipTextWithLineBreaks(teamNames)}
                underline={false}
                showArrow
                position="top"
                tipOffset={10}
                fixedPositionStrategy
              >
                {teamsText}
              </TooltipWrapper>
            ) : (
              teamsText
            )
          }
        />
      )}
      <DataSet
        title="Role"
        value={
          roleText === "Various" ? (
            <TooltipWrapper
              tipContent={roleGroups.map(({ role, names }) => (
                <span key={role}>
                  <b>{role}:</b> {names.join(", ")}
                  <br />
                </span>
              ))}
              underline={false}
              showArrow
              position="top"
              tipOffset={10}
              fixedPositionStrategy
            >
              {roleText}
            </TooltipWrapper>
          ) : (
            roleText
          )
        }
      />
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
