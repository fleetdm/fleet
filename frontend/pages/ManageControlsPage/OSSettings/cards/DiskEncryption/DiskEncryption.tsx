import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import { ITeamConfig } from "interfaces/team";
import {
  DISK_ENCRYPTION_SETTINGS_PLATFORMS,
  DiskEncryptionSettingsPlatform,
  isDiskEncryptionSettingsPlatform,
} from "interfaces/platform";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import permissions from "utilities/permissions";

import diskEncryptionAPI, {
  IUpdateDiskEncryptionFormData,
} from "services/entities/disk_encryption";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";

import Button from "components/buttons/Button";
import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import DiskEncryptionTable from "./components/DiskEncryptionTable";
import {
  getDiskEncryptionSettings,
  getErrorMessage,
  IDiskEncryptionSettings,
  isMacOSDiskEncryptionEnforceOnly,
} from "./helpers";
import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

const baseClass = "disk-encryption";

const PLATFORM_TAB_NAMES: Record<DiskEncryptionSettingsPlatform, string> = {
  macos: "macOS",
  windows: "Windows",
  linux: "Linux",
};

const getPlatformTabPath = (
  platform: DiskEncryptionSettingsPlatform,
  teamId: number
) =>
  getPathWithQueryParams(PATHS.CONTROLS_DISK_ENCRYPTION_PLATFORM(platform), {
    fleet_id: teamId,
  });

const MDM_REQUIRED_EMPTY_STATES: Partial<
  Record<DiskEncryptionSettingsPlatform, JSX.Element>
> = {
  macos: (
    <EmptyState
      header="Turn on MDM to enforce disk encryption"
      info={
        <>
          You must turn on Apple MDM to enforce disk encryption for macOS hosts.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/turn-on-apple-mdm`}
            text="Learn more"
            newTab
          />
        </>
      }
      variant="form"
    />
  ),
  windows: (
    <EmptyState
      header="Turn on MDM to enforce disk encryption"
      info={
        <>
          You must turn on Windows MDM to enforce disk encryption for Windows
          hosts.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-windows-mdm`}
            text="Learn more"
            newTab
          />
        </>
      }
      variant="form"
    />
  ),
};

export type IDiskEncryptionProps = IOSSettingsCommonProps;

const DiskEncryption = ({
  currentTeamId,
  onMutation,
  router,
  urlPlatformParam,
}: IDiskEncryptionProps) => {
  const {
    isPremiumTier,
    config,
    setConfig,
    isTeamTechnician,
    isGlobalTechnician,
  } = useContext(AppContext);

  const isTechnician = isTeamTechnician || isGlobalTechnician;
  const gitOpsModeEnabled = config?.gitops?.gitops_mode_enabled;

  const selectedPlatform = isDiskEncryptionSettingsPlatform(urlPlatformParam)
    ? urlPlatformParam
    : undefined;

  useEffect(() => {
    if (!selectedPlatform) {
      router.replace(getPlatformTabPath("macos", currentTeamId));
    }
  }, [selectedPlatform, router, currentTeamId]);

  const onSelectPlatformTab = (index: number) => {
    router.push(
      getPlatformTabPath(
        DISK_ENCRYPTION_SETTINGS_PLATFORMS[index],
        currentTeamId
      )
    );
  };

  const [isSaving, setIsSaving] = useState(false);

  // Linux escrow is handled by fleetd, so it doesn't depend on MDM
  const isPlatformMdmEnabled: Record<
    DiskEncryptionSettingsPlatform,
    boolean
  > = {
    macos: !!config && permissions.isMacMdmEnabledAndConfigured(config),
    windows: !!config && permissions.isWindowsMdmEnabledAndConfigured(config),
    linux: true,
  };

  const isPlatformFormDisabled = (platform: DiskEncryptionSettingsPlatform) =>
    gitOpsModeEnabled || isSaving || !isPlatformMdmEnabled[platform];
  const [formSettings, setFormSettings] = useState<IDiskEncryptionSettings>(
    () =>
      currentTeamId === 0
        ? getDiskEncryptionSettings(config?.mdm)
        : getDiskEncryptionSettings()
  );
  // the settings as last persisted, which determine whether each platform's
  // status table is shown
  const [savedSettings, setSavedSettings] = useState<IDiskEncryptionSettings>(
    formSettings
  );

  // because we pull the default state for no teams from the config,
  // we need to update the config when the user toggles the checkbox
  const getUpdatedAppConfig = async () => {
    try {
      const updatedConfig = await configAPI.loadAll();
      setConfig(updatedConfig);
    } catch (err) {
      notify.error("Could not retrieve updated app config. Please try again.", {
        response: err,
      });
    }
  };

  const { isLoading: isLoadingTeam, isError: isTeamError } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    refetchOnWindowFocus: false,
    retry: false,
    enabled: currentTeamId !== 0,
    select: (res) => res.fleet,
    onSuccess: (res) => {
      const settings = getDiskEncryptionSettings(res?.mdm);
      setFormSettings(settings);
      setSavedSettings(settings);
    },
  });

  const onToggleSetting = (key: keyof IDiskEncryptionSettings) => (
    value: boolean
  ) => {
    setFormSettings((prev) => {
      const next = { ...prev, [key]: value };
      // a BitLocker PIN can't be required while Windows disk encryption is off
      if (key === "windowsEnabled" && !value) {
        next.windowsPINRequired = false;
      }
      return next;
    });
  };

  const onSaveDiskEncryption = async (
    platform: DiskEncryptionSettingsPlatform
  ) => {
    if (isSaving) return;

    let formData: IUpdateDiskEncryptionFormData;
    let updatedSettings: Partial<IDiskEncryptionSettings>;
    switch (platform) {
      case "macos":
        formData = {
          macos_settings: {
            enable_disk_encryption: formSettings.macOSEnabled,
            enable_escrow_disk_encryption_key: formSettings.macOSEscrowEnabled,
          },
        };
        updatedSettings = {
          macOSEnabled: formSettings.macOSEnabled,
          macOSEscrowEnabled: formSettings.macOSEscrowEnabled,
        };
        break;
      case "windows":
        formData = {
          windows_settings: {
            enable_disk_encryption: formSettings.windowsEnabled,
            require_bitlocker_pin: formSettings.windowsPINRequired,
          },
        };
        updatedSettings = {
          windowsEnabled: formSettings.windowsEnabled,
          windowsPINRequired: formSettings.windowsPINRequired,
        };
        break;
      case "linux":
      default:
        formData = {
          linux_settings: {
            enable_escrow_disk_encryption_key: formSettings.linuxEscrowEnabled,
          },
        };
        updatedSettings = {
          linuxEscrowEnabled: formSettings.linuxEscrowEnabled,
        };
    }

    setIsSaving(true);
    try {
      await diskEncryptionAPI.updateDiskEncryption(formData, currentTeamId);
      notify.success("Successfully updated disk encryption settings.");
      onMutation();
      setSavedSettings((prev) => ({ ...prev, ...updatedSettings }));
      if (currentTeamId === 0) {
        getUpdatedAppConfig();
      }
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
    } finally {
      setIsSaving(false);
    }
  };

  const renderSaveButton = (platform: DiskEncryptionSettingsPlatform) => (
    <GitOpsModeTooltipWrapper
      renderChildren={(disableChildren) => (
        <Button
          disabled={disableChildren || isPlatformFormDisabled(platform)}
          isLoading={isSaving}
          className={`${baseClass}__save-button`}
          onClick={() => onSaveDiskEncryption(platform)}
        >
          Save
        </Button>
      )}
    />
  );

  const renderEnforceCheckbox = (
    platform: DiskEncryptionSettingsPlatform,
    key: "macOSEnabled" | "windowsEnabled",
    value: boolean
  ) => (
    <Checkbox
      disabled={isPlatformFormDisabled(platform)}
      onChange={onToggleSetting(key)}
      value={value}
      className={`${baseClass}__checkbox`}
      helpText={
        <>
          If turned on, Fleet enforces disk encryption.{" "}
          <CustomLink
            text="Learn more"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/mdm-disk-encryption`}
            newTab
          />
        </>
      }
    >
      Enable disk encryption
    </Checkbox>
  );

  const renderEscrowCheckbox = (
    platform: DiskEncryptionSettingsPlatform,
    key: "macOSEscrowEnabled" | "linuxEscrowEnabled",
    value: boolean,
    learnMoreLink?: string
  ) => (
    <Checkbox
      disabled={isPlatformFormDisabled(platform)}
      onChange={onToggleSetting(key)}
      value={value}
      className={`${baseClass}__checkbox`}
      helpText={
        <>
          Store the recovery key so your fleet can recover the device if the end
          user forgets their password.
          {learnMoreLink && (
            <>
              {" "}
              <CustomLink text="Learn more" url={learnMoreLink} newTab />
            </>
          )}
        </>
      }
    >
      Escrow recovery key with Fleet
    </Checkbox>
  );

  const renderPlatformTabPanel = (
    platform: DiskEncryptionSettingsPlatform,
    isEnabled: boolean,
    formFields: JSX.Element
  ) => {
    // GitOps mode has its own tooltip on Save, so only show the MDM-required
    // empty state when that isn't what's disabling the form
    const mdmRequiredEmptyState =
      gitOpsModeEnabled || isPlatformMdmEnabled[platform]
        ? undefined
        : MDM_REQUIRED_EMPTY_STATES[platform];

    return (
      <>
        {!isTechnician &&
          (mdmRequiredEmptyState || (
            <Card
              className={`${baseClass}__settings-card`}
              color="white"
              borderRadiusSize="large"
            >
              <div className={`${baseClass}__form-fields`}>{formFields}</div>
              {renderSaveButton(platform)}
            </Card>
          ))}
        {isEnabled ? (
          <DiskEncryptionTable
            platform={platform}
            currentTeamId={currentTeamId}
            isMacOSEnforceOnly={
              platform === "macos" &&
              isMacOSDiskEncryptionEnforceOnly(savedSettings)
            }
            router={router}
          />
        ) : (
          isTechnician && <p>Disk encryption is disabled.</p>
        )}
      </>
    );
  };

  const renderContent = () => {
    if (isTeamError) {
      return <DataError />;
    }

    // no selected platform means the redirect to the macOS tab is in flight
    if (isLoadingTeam || !selectedPlatform) {
      return <Spinner />;
    }

    return (
      <TabNav secondary>
        <Tabs
          selectedIndex={DISK_ENCRYPTION_SETTINGS_PLATFORMS.indexOf(
            selectedPlatform
          )}
          onSelect={onSelectPlatformTab}
        >
          <TabList>
            {DISK_ENCRYPTION_SETTINGS_PLATFORMS.map((platform) => (
              <Tab key={platform}>
                <TabText>{PLATFORM_TAB_NAMES[platform]}</TabText>
              </Tab>
            ))}
          </TabList>
          <TabPanel className={`${baseClass}__tab-panel`}>
            {renderPlatformTabPanel(
              "macos",
              savedSettings.macOSEnabled || savedSettings.macOSEscrowEnabled,
              <>
                {renderEnforceCheckbox(
                  "macos",
                  "macOSEnabled",
                  formSettings.macOSEnabled
                )}
                {renderEscrowCheckbox(
                  "macos",
                  "macOSEscrowEnabled",
                  formSettings.macOSEscrowEnabled
                )}
              </>
            )}
          </TabPanel>
          <TabPanel className={`${baseClass}__tab-panel`}>
            {renderPlatformTabPanel(
              "windows",
              savedSettings.windowsEnabled,
              <>
                {renderEnforceCheckbox(
                  "windows",
                  "windowsEnabled",
                  formSettings.windowsEnabled
                )}
                <Checkbox
                  disabled={
                    isPlatformFormDisabled("windows") ||
                    !formSettings.windowsEnabled
                  }
                  onChange={onToggleSetting("windowsPINRequired")}
                  value={formSettings.windowsPINRequired}
                  className={`${baseClass}__checkbox`}
                >
                  <TooltipWrapper
                    tipContent={
                      <div>
                        <>
                          If enabled, end users on Windows hosts will be
                          required to set a BitLocker PIN.
                          <br />
                          When the PIN is set, it&rsquo;s required to unlock
                          Windows hosts during startup.
                        </>
                      </div>
                    }
                  >
                    Require BitLocker PIN
                  </TooltipWrapper>
                </Checkbox>
              </>
            )}
          </TabPanel>
          <TabPanel className={`${baseClass}__tab-panel`}>
            {renderPlatformTabPanel(
              "linux",
              savedSettings.linuxEscrowEnabled,
              renderEscrowCheckbox(
                "linux",
                "linuxEscrowEnabled",
                formSettings.linuxEscrowEnabled,
                `${LEARN_MORE_ABOUT_BASE_LINK}/linux-disk-encryption`
              )
            )}
          </TabPanel>
        </Tabs>
      </TabNav>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Disk encryption" alignLeftHeaderVertically />
      {isPremiumTier ? renderContent() : <PremiumFeatureMessage />}
    </div>
  );
};

export default DiskEncryption;
