import React, { useContext, useEffect, useState } from "react";

import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { EndUserLocalAccountType } from "interfaces/mdm";

import EndUserAuthSection from "./components/EndUserAuthSection";
import LocalAccountSection, {
  effectiveEnableManagedLocalAccount,
} from "./components/LocalAccountSection/LocalAccountSection";

const baseClass = "users-form";

export interface IUsersFormData {
  endUserAuthEnabled: boolean;
  lockEndUserInfo: boolean;
  enableManagedLocalAccount: boolean;
  localAccountType: EndUserLocalAccountType;
}

interface IUsersFormProps {
  currentTeamId: number;
  defaultIsEndUserAuthEnabled: boolean;
  defaultLockEndUserInfo: boolean;
  defaultEnableManagedLocalAccount: boolean;
  /** The radio value to start from. Defaults to the option that doesn't
   * force the managed local account on. */
  defaultLocalAccountType?: EndUserLocalAccountType;
  isIdPConfigured: boolean;
}

const UsersForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
  defaultLockEndUserInfo,
  defaultEnableManagedLocalAccount,
  defaultLocalAccountType = EndUserLocalAccountType.ADMIN,
  isIdPConfigured,
}: IUsersFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config, isMacMdmEnabledAndConfigured } = useContext(AppContext);
  const gitOpsModeEnabled = !!config?.gitops.gitops_mode_enabled;

  const [formData, setFormData] = useState<IUsersFormData>({
    endUserAuthEnabled: defaultIsEndUserAuthEnabled,
    lockEndUserInfo: defaultLockEndUserInfo,
    enableManagedLocalAccount: defaultEnableManagedLocalAccount,
    localAccountType: defaultLocalAccountType,
  });
  const [isUpdating, setIsUpdating] = useState(false);

  // Re-sync local state when the parent refetches config (e.g. team switch).
  // useState initializers only run on first mount, so without this the form
  // would show and save the previous team's settings.
  useEffect(() => {
    setFormData({
      endUserAuthEnabled: defaultIsEndUserAuthEnabled,
      lockEndUserInfo: defaultLockEndUserInfo,
      enableManagedLocalAccount: defaultEnableManagedLocalAccount,
      localAccountType: defaultLocalAccountType,
    });
  }, [
    defaultIsEndUserAuthEnabled,
    defaultLockEndUserInfo,
    defaultEnableManagedLocalAccount,
    defaultLocalAccountType,
  ]);

  const onEndUserAuthChange = (value: boolean) => {
    // Sync lock end user info with EUA only when Apple MDM is configured.
    // Without Apple MDM the field is read-only, so we leave whatever value
    // came from the backend untouched.
    setFormData((prev) => ({
      ...prev,
      endUserAuthEnabled: value,
      lockEndUserInfo: isMacMdmEnabledAndConfigured
        ? value
        : prev.lockEndUserInfo,
    }));
  };

  const onLockEndUserInfoChange = (value: boolean) => {
    setFormData((prev) => ({ ...prev, lockEndUserInfo: value }));
  };

  const onEnableManagedLocalAccountChange = (value: boolean) => {
    setFormData((prev) => ({ ...prev, enableManagedLocalAccount: value }));
  };

  const onLocalAccountTypeChange = (value: EndUserLocalAccountType) => {
    setFormData((prev) => ({ ...prev, localAccountType: value }));
  };

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    setIsUpdating(true);
    const canLockEndUserInfo =
      formData.endUserAuthEnabled && formData.lockEndUserInfo;

    try {
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        enable_end_user_authentication: formData.endUserAuthEnabled,
        // Apple-only fields are omitted when Apple MDM isn't configured.
        ...(isMacMdmEnabledAndConfigured && {
          lock_end_user_info: canLockEndUserInfo,
          enable_managed_local_account: effectiveEnableManagedLocalAccount(
            formData
          ),
          end_user_local_account_type: formData.localAccountType,
        }),
      });
      renderFlash("success", "Successfully updated.");
    } catch {
      renderFlash("error", "Couldn't update settings. Please try again.");
    }

    setIsUpdating(false);
    if (isMacMdmEnabledAndConfigured) {
      setFormData((prev) => ({ ...prev, lockEndUserInfo: canLockEndUserInfo }));
    }
  };

  return (
    <div className={baseClass}>
      <form onSubmit={onSubmit}>
        <LocalAccountSection
          formData={formData}
          onLocalAccountTypeChange={onLocalAccountTypeChange}
          onEnableManagedLocalAccountChange={onEnableManagedLocalAccountChange}
          isMacMdmEnabledAndConfigured={!!isMacMdmEnabledAndConfigured}
        />
        <EndUserAuthSection
          endUserAuthEnabled={formData.endUserAuthEnabled}
          lockEndUserInfo={formData.lockEndUserInfo}
          onEndUserAuthChange={onEndUserAuthChange}
          onLockEndUserInfoChange={onLockEndUserInfoChange}
          isIdPConfigured={isIdPConfigured}
          isMacMdmEnabledAndConfigured={!!isMacMdmEnabledAndConfigured}
          gitOpsModeEnabled={gitOpsModeEnabled}
        />
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
