import React, { useContext, useEffect, useState } from "react";
import { useQueryClient } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import Button from "components/buttons/Button";
import Dropdown from "components/forms/fields/Dropdown";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { IConfig } from "interfaces/config";
import { EndUserLocalAccountType } from "interfaces/mdm";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import configAPI from "services/entities/config";
import mdmAPI from "services/entities/mdm";
import teamsAPI from "services/entities/teams";
import { resolvedEndUserIdPConfigured } from "utilities/permissions/permissions";

import EndUserAuthSection from "./components/EndUserAuthSection";
import LocalAccountSection, {
  effectiveEnableManagedLocalAccount,
} from "./components/LocalAccountSection/LocalAccountSection";
import WindowsAccountSection from "./components/WindowsAccountSection";

const baseClass = "users-form";

export interface IUsersFormData {
  endUserAuthEnabled: boolean;
  lockEndUserInfo: boolean;
  enableManagedLocalAccount: boolean;
  localAccountType: EndUserLocalAccountType;
  enableManagedLocalAccountWindows: boolean;
  identityProvider: string;
}

interface IUsersFormProps {
  currentTeamId: number;
  defaultIsEndUserAuthEnabled: boolean;
  defaultLockEndUserInfo: boolean;
  defaultEnableManagedLocalAccount: boolean;
  /** The radio value to start from. Defaults to the option that doesn't
   * force the managed local account on. */
  defaultLocalAccountType?: EndUserLocalAccountType;
  defaultEnableManagedLocalAccountWindows: boolean;
  isIdPConfigured: boolean;
  globalConfig?: IConfig;
  defaultIdentityProvider?: string;
}

const UsersForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
  defaultLockEndUserInfo,
  defaultEnableManagedLocalAccount,
  defaultLocalAccountType = EndUserLocalAccountType.ADMIN,
  defaultEnableManagedLocalAccountWindows,
  isIdPConfigured,
  globalConfig,
  defaultIdentityProvider = "",
}: IUsersFormProps) => {
  const {
    config,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
  } = useContext(AppContext);
  const gitOpsModeEnabled = !!config?.gitops.gitops_mode_enabled;
  const queryClient = useQueryClient();

  const [formData, setFormData] = useState<IUsersFormData>({
    endUserAuthEnabled: defaultIsEndUserAuthEnabled,
    lockEndUserInfo: defaultLockEndUserInfo,
    enableManagedLocalAccount: defaultEnableManagedLocalAccount,
    localAccountType: defaultLocalAccountType,
    enableManagedLocalAccountWindows: defaultEnableManagedLocalAccountWindows,
    identityProvider: defaultIdentityProvider,
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
      enableManagedLocalAccountWindows: defaultEnableManagedLocalAccountWindows,
      identityProvider: defaultIdentityProvider,
    });
  }, [
    defaultIsEndUserAuthEnabled,
    defaultLockEndUserInfo,
    defaultEnableManagedLocalAccount,
    defaultLocalAccountType,
    defaultEnableManagedLocalAccountWindows,
    defaultIdentityProvider,
  ]);

  const orgDefaultValue = "__org_default__";
  const identityProviders = globalConfig?.mdm.identity_providers ?? [];
  const showIdentityProvider =
    currentTeamId !== APP_CONTEXT_NO_TEAM_ID && identityProviders.length > 0;
  const idpReady = globalConfig
    ? resolvedEndUserIdPConfigured(
        globalConfig,
        currentTeamId === APP_CONTEXT_NO_TEAM_ID
          ? ""
          : formData.identityProvider
      )
    : isIdPConfigured;

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

  const onEnableManagedLocalAccountWindowsChange = (value: boolean) => {
    setFormData((prev) => ({
      ...prev,
      enableManagedLocalAccountWindows: value,
    }));
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
      if (showIdentityProvider) {
        await teamsAPI.updateConfig(
          { mdm: { identity_provider: formData.identityProvider } },
          currentTeamId
        );
      }

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

      // The Windows toggle isn't part of the Apple Setup Assistant flow that /setup_experience models, so it saves
      // through the MDM config instead. Skipped entirely when Windows MDM is off.
      if (isWindowsMdmEnabledAndConfigured) {
        const mdmUpdate = {
          windows_settings: {
            enable_managed_local_account:
              formData.enableManagedLocalAccountWindows,
          },
        };
        if (currentTeamId === APP_CONTEXT_NO_TEAM_ID) {
          await configAPI.update({ mdm: mdmUpdate });
        } else {
          await teamsAPI.updateConfig({ mdm: mdmUpdate }, currentTeamId);
        }
      }

      // Both calls above write into the app config and the fleet, which several other cards read
      // from the same cache keys, so drop them.
      await queryClient.invalidateQueries(["config"]);
      if (currentTeamId !== APP_CONTEXT_NO_TEAM_ID) {
        await queryClient.invalidateQueries(["team", currentTeamId]);
      }

      notify.success("Successfully updated.");
    } catch (err) {
      notify.error("Couldn't update settings. Please try again.", {
        response: err,
      });
    }

    setIsUpdating(false);
    if (isMacMdmEnabledAndConfigured) {
      setFormData((prev) => ({ ...prev, lockEndUserInfo: canLockEndUserInfo }));
    }
  };

  return (
    <div className={baseClass}>
      <form onSubmit={onSubmit}>
        {showIdentityProvider && (
          <Dropdown
            label="Identity provider"
            name="identity-provider"
            options={[
              { label: "Organization default", value: orgDefaultValue },
              ...identityProviders.map((provider) => ({
                label: provider.default
                  ? `${provider.name} (organization default)`
                  : provider.name,
                value: provider.name,
              })),
            ]}
            value={formData.identityProvider || orgDefaultValue}
            onChange={(value: string) =>
              setFormData((prev) => ({
                ...prev,
                identityProvider:
                  !value || value === orgDefaultValue ? "" : value,
              }))
            }
            disabled={gitOpsModeEnabled}
            helpText="This fleet uses the selected identity provider for end-user sign-in. Organization default uses the connection marked default, or the legacy end-user authentication settings."
          />
        )}
        <EndUserAuthSection
          endUserAuthEnabled={formData.endUserAuthEnabled}
          lockEndUserInfo={formData.lockEndUserInfo}
          onEndUserAuthChange={onEndUserAuthChange}
          onLockEndUserInfoChange={onLockEndUserInfoChange}
          isIdPConfigured={idpReady}
          isMacMdmEnabledAndConfigured={!!isMacMdmEnabledAndConfigured}
          gitOpsModeEnabled={gitOpsModeEnabled}
        />
        <TabNav secondary>
          <Tabs>
            <TabList>
              <Tab>
                <TabText>macOS</TabText>
              </Tab>
              <Tab>
                <TabText>Windows</TabText>
              </Tab>
            </TabList>
            <TabPanel>
              <LocalAccountSection
                formData={formData}
                onLocalAccountTypeChange={onLocalAccountTypeChange}
                onEnableManagedLocalAccountChange={
                  onEnableManagedLocalAccountChange
                }
                isMacMdmEnabledAndConfigured={!!isMacMdmEnabledAndConfigured}
              />
            </TabPanel>
            <TabPanel>
              <WindowsAccountSection
                enableManagedLocalAccount={
                  formData.enableManagedLocalAccountWindows
                }
                onEnableManagedLocalAccountChange={
                  onEnableManagedLocalAccountWindowsChange
                }
                isWindowsMdmEnabledAndConfigured={
                  !!isWindowsMdmEnabledAndConfigured
                }
              />
            </TabPanel>
          </Tabs>
        </TabNav>
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
