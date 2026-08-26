import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import { ITeamConfig } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import diskEncryptionAPI, {
  IUpdateDiskEncryptionFormData,
} from "services/entities/disk_encryption";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";

import Button from "components/buttons/Button";
import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import DiskEncryptionTable from "./components/DiskEncryptionTable";
import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

const baseClass = "disk-encryption";

type DiskEncryptionPlatform = "macos" | "windows" | "linux";

interface IDiskEncryptionSettings {
  macOSEnabled: boolean;
  macOSEscrowEnabled: boolean;
  windowsEnabled: boolean;
  windowsPINRequired: boolean;
  linuxEscrowEnabled: boolean;
}

/** Structural subset of the global (`IMdmConfig`) and fleet
 * (`ITeamConfig["mdm"]`) mdm config shapes this card reads. */
interface IMdmDiskEncryptionSource {
  enable_disk_encryption?: boolean;
  windows_require_bitlocker_pin?: boolean;
  apple_settings?: {
    enable_disk_encryption?: boolean;
    enable_escrow_disk_encryption_key?: boolean;
  };
  windows_settings?: {
    enable_disk_encryption?: boolean;
    require_bitlocker_pin?: boolean;
  };
  linux_settings?: {
    enable_escrow_disk_encryption_key?: boolean;
  };
}

const getDiskEncryptionSettings = (
  mdm?: IMdmDiskEncryptionSource
): IDiskEncryptionSettings => {
  // fall back to the deprecated top-level keys so configs from servers that
  // don't return the per-platform fields yet still render their effective
  // state
  const legacyEnabled = mdm?.enable_disk_encryption ?? false;
  return {
    macOSEnabled: mdm?.apple_settings?.enable_disk_encryption ?? legacyEnabled,
    macOSEscrowEnabled:
      mdm?.apple_settings?.enable_escrow_disk_encryption_key ?? legacyEnabled,
    windowsEnabled:
      mdm?.windows_settings?.enable_disk_encryption ?? legacyEnabled,
    windowsPINRequired:
      mdm?.windows_settings?.require_bitlocker_pin ??
      mdm?.windows_require_bitlocker_pin ??
      false,
    linuxEscrowEnabled:
      mdm?.linux_settings?.enable_escrow_disk_encryption_key ?? legacyEnabled,
  };
};

const PLATFORM_BY_INDEX: DiskEncryptionPlatform[] = [
  "macos",
  "windows",
  "linux",
];

const PLATFORM_TAB_NAMES: Record<DiskEncryptionPlatform, string> = {
  macos: "macOS",
  windows: "Windows",
  linux: "Linux",
};

export type IDiskEncryptionProps = IOSSettingsCommonProps;

const DiskEncryption = ({
  currentTeamId,
  onMutation,
  router,
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

  const [isLoadingTeam, setIsLoadingTeam] = useState(currentTeamId !== 0);
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

  useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: currentTeamId !== 0,
      select: (res) => res.fleet,
      onSuccess: (res) => {
        const settings = getDiskEncryptionSettings(res.mdm);
        setFormSettings(settings);
        setSavedSettings(settings);
        setIsLoadingTeam(false);
      },
    }
  );

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

  const onSaveDiskEncryption = async (platform: DiskEncryptionPlatform) => {
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

    try {
      await diskEncryptionAPI.updateDiskEncryption(formData, currentTeamId);
      notify.success("Successfully updated disk encryption settings.");
      onMutation();
      setSavedSettings((prev) => ({ ...prev, ...updatedSettings }));
      if (currentTeamId === 0) {
        getUpdatedAppConfig();
      }
    } catch (e) {
      if (getErrorReason(e).includes("Missing required private key")) {
        const link =
          "https://fleetdm.com/learn-more-about/fleet-server-private-key";
        notify.error(
          <>
            Couldn&apos;t enable disk encryption. Please configure a private
            key.{" "}
            <CustomLink
              url={link}
              text="Learn how"
              newTab
              variant="flash-message-link"
            />
          </>,
          { response: e }
        );
      } else {
        const errorMsg =
          getErrorReason(e) ??
          "Could not update the disk encryption settings. Please try again.";
        notify.error(errorMsg, { response: e });
      }
    }
  };

  const renderSaveButton = (platform: DiskEncryptionPlatform) => (
    <GitOpsModeTooltipWrapper
      renderChildren={(disableChildren) => (
        <Button
          disabled={disableChildren}
          className={`${baseClass}__save-button`}
          onClick={() => onSaveDiskEncryption(platform)}
        >
          Save
        </Button>
      )}
    />
  );

  const enforceCheckbox = (
    key: "macOSEnabled" | "windowsEnabled",
    value: boolean
  ) => (
    <Checkbox
      disabled={gitOpsModeEnabled}
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

  const escrowCheckbox = (
    key: "macOSEscrowEnabled" | "linuxEscrowEnabled",
    value: boolean,
    learnMoreLink?: string
  ) => (
    <Checkbox
      disabled={gitOpsModeEnabled}
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
    platform: DiskEncryptionPlatform,
    isEnabled: boolean,
    formFields: JSX.Element
  ) => (
    <>
      {!isTechnician && (
        <Card
          className={`${baseClass}__settings-card`}
          color="white"
          borderRadiusSize="large"
        >
          <div className={`${baseClass}__form-fields`}>{formFields}</div>
          {renderSaveButton(platform)}
        </Card>
      )}
      {isEnabled ? (
        <DiskEncryptionTable
          platform={platform}
          currentTeamId={currentTeamId}
          router={router}
        />
      ) : (
        isTechnician && <p>Disk encryption is disabled.</p>
      )}
    </>
  );

  const renderContent = () => {
    if (isLoadingTeam) {
      return <Spinner />;
    }

    return (
      <TabNav secondary>
        <Tabs>
          <TabList>
            {PLATFORM_BY_INDEX.map((platform) => (
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
                {enforceCheckbox("macOSEnabled", formSettings.macOSEnabled)}
                {escrowCheckbox(
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
                {enforceCheckbox("windowsEnabled", formSettings.windowsEnabled)}
                <Checkbox
                  disabled={gitOpsModeEnabled || !formSettings.windowsEnabled}
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
              escrowCheckbox(
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
