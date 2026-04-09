import React, { useContext, useState } from "react";

import PATHS from "router/paths";
import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "users-form";

interface IUsersFormProps {
  currentTeamId: number;
  defaultIsEndUserAuthEnabled: boolean;
  defaultLockEndUserInfo: boolean;
  defaultEnableManagedLocalAccount: boolean;
}

const UsersForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
  defaultLockEndUserInfo,
  defaultEnableManagedLocalAccount,
}: IUsersFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [isEndUserAuthEnabled, setEndUserAuthEnabled] = useState(
    defaultIsEndUserAuthEnabled
  );
  const [lockEndUserInfo, setLockEndUserInfo] = useState(
    defaultLockEndUserInfo
  );
  const [enableManagedLocalAccount, setEnableManagedLocalAccount] = useState(
    defaultEnableManagedLocalAccount
  );
  const [isUpdating, setIsUpdating] = useState(false);

  const onToggleEndUserAuth = (newCheckVal: boolean) => {
    setEndUserAuthEnabled(newCheckVal);
    // Sync lock end user info with EUA: enabling EUA enables it, disabling EUA disables it.
    setLockEndUserInfo(newCheckVal);
  };

  const onChangeLockEndUserInfo = (newCheckVal: boolean) => {
    setLockEndUserInfo(newCheckVal);
  };

  const onToggleManagedLocalAccount = (newCheckVal: boolean) => {
    setEnableManagedLocalAccount(newCheckVal);
  };

  const onClickSave = async () => {
    setIsUpdating(true);
    const canLockEndUserInfo = isEndUserAuthEnabled && lockEndUserInfo;
    try {
      // Save both end user auth and managed local account settings together
      await mdmAPI.updateEndUserAuthentication(
        currentTeamId,
        isEndUserAuthEnabled,
        canLockEndUserInfo
      );
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        enable_managed_local_account: enableManagedLocalAccount,
      });
      renderFlash("success", "Successfully updated.");
    } catch {
      renderFlash("error", "Couldn't update. Please try again.");
    } finally {
      setIsUpdating(false);
      setLockEndUserInfo(canLockEndUserInfo);
    }
  };

  return (
    <div className={baseClass}>
      <form>
        <Checkbox
          disabled={gitOpsModeEnabled}
          value={isEndUserAuthEnabled}
          onChange={onToggleEndUserAuth}
          helpText={
            <span>
              End users are required to authenticate with your{" "}
              <CustomLink
                url={PATHS.ADMIN_INTEGRATIONS_SSO_END_USERS}
                text="identity provider (IdP)"
              />{" "}
              when setting up new hosts.
            </span>
          }
        >
          End user authentication
        </Checkbox>
        {isEndUserAuthEnabled && (
          <div className={`${baseClass}__advanced-options`}>
            <Checkbox
              disabled={gitOpsModeEnabled}
              onChange={onChangeLockEndUserInfo}
              value={lockEndUserInfo}
            >
              <TooltipWrapper
                tipContent={
                  <span>
                    End user can&apos;t edit the local account&apos;s{" "}
                    <b>Account Name</b> and
                    <br />
                    <b>Full Name</b> in macOS Setup Assistant. These fields will
                    be
                    <br />
                    locked to values from your IdP.
                  </span>
                }
              >
                Lock end user info
              </TooltipWrapper>
            </Checkbox>
          </div>
        )}
        <Checkbox
          disabled={gitOpsModeEnabled}
          value={enableManagedLocalAccount}
          onChange={onToggleManagedLocalAccount}
          helpText={
            <span>
              Fleet generates a user (_fleetadmin) and unique password for each
              host, accessible in <b>Host details</b> &gt;{" "}
              <b>Show managed account</b>.
            </span>
          }
        >
          <TooltipWrapper tipContent="Creates a hidden managed local admin account for remote troubleshooting on macOS hosts.">
            Managed local account
          </TooltipWrapper>
        </Checkbox>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              isLoading={isUpdating}
              onClick={onClickSave}
            >
              Save
            </Button>
          )}
        />
      </form>
    </div>
  );
};

export default UsersForm;
