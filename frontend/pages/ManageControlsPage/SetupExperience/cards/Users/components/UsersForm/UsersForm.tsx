import React, { useContext, useEffect, useState } from "react";

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
  isIdPConfigured: boolean;
}

const UsersForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
  defaultLockEndUserInfo,
  defaultEnableManagedLocalAccount,
  isIdPConfigured,
}: IUsersFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config, isMacMdmEnabledAndConfigured } = useContext(AppContext);
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

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

  // Re-sync local state when the parent refetches config (e.g. team switch).
  // useState initializers only run on first mount, so without this the form
  // would show and save the previous team's settings.
  useEffect(() => {
    setEndUserAuthEnabled(defaultIsEndUserAuthEnabled);
    setLockEndUserInfo(defaultLockEndUserInfo);
    setEnableManagedLocalAccount(defaultEnableManagedLocalAccount);
  }, [
    defaultIsEndUserAuthEnabled,
    defaultLockEndUserInfo,
    defaultEnableManagedLocalAccount,
  ]);

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

  const onClickSave = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsUpdating(true);
    const canLockEndUserInfo = isEndUserAuthEnabled && lockEndUserInfo;

    try {
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        enable_end_user_authentication: isEndUserAuthEnabled,
        lock_end_user_info: canLockEndUserInfo,
        enable_managed_local_account: enableManagedLocalAccount,
      });
      renderFlash("success", "Successfully updated.");
    } catch {
      renderFlash("error", "Couldn't update settings. Please try again.");
    }

    setIsUpdating(false);
    setLockEndUserInfo(canLockEndUserInfo);
  };

  return (
    <div className={baseClass}>
      <form onSubmit={onClickSave}>
        <TooltipWrapper
          tipContent={
            !isIdPConfigured ? (
              <span>
                To enable, first connect Fleet to
                <br />
                your{" "}
                <CustomLink
                  url={PATHS.ADMIN_INTEGRATIONS_SSO_END_USERS}
                  text="identity provider (IdP)"
                  variant="tooltip-link"
                />
                .
              </span>
            ) : undefined
          }
          disableTooltip={isIdPConfigured}
          underline={false}
          position="left"
          showArrow
        >
          <Checkbox
            disabled={gitOpsModeEnabled || !isIdPConfigured}
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
        </TooltipWrapper>
        {isEndUserAuthEnabled && (
          <div className={`${baseClass}__advanced-options`}>
            <Checkbox
              disabled={gitOpsModeEnabled || !isIdPConfigured}
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
        <TooltipWrapper
          tipContent={
            !isMacMdmEnabledAndConfigured ? (
              <span>
                To enable, first turn on{" "}
                <CustomLink
                  url={PATHS.ADMIN_INTEGRATIONS_MDM_APPLE}
                  text="Apple MDM"
                  variant="tooltip-link"
                />
                .
              </span>
            ) : undefined
          }
          disableTooltip={!!isMacMdmEnabledAndConfigured}
          underline={false}
          position="left"
          showArrow
        >
          <Checkbox
            disabled={gitOpsModeEnabled || !isMacMdmEnabledAndConfigured}
            value={enableManagedLocalAccount}
            onChange={onToggleManagedLocalAccount}
            helpText={
              <span>
                Fleet generates a user (_fleetadmin) and unique password for
                each host, accessible in <b>Host details</b> &gt;{" "}
                <b>Show managed account</b>.
              </span>
            }
          >
            <TooltipWrapper
              tipContent={
                <>
                  Creates a hidden managed local admin account for
                  <br />
                  remote troubleshooting on macOS hosts.
                </>
              }
            >
              Managed local account
            </TooltipWrapper>
          </Checkbox>
        </TooltipWrapper>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              isLoading={isUpdating}
              type="submit"
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
