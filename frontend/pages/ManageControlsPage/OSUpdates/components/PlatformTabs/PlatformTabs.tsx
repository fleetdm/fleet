import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import { LEARN_MORE_ABOUT_BASE_LINK, SUPPORT_LINK } from "utilities/constants";

import EndUserOSRequirementPreview from "../EndUserOSRequirementPreview";
import WindowsTargetForm from "../WindowsTargetForm";
import { OSUpdatesTargetPlatform } from "../../OSUpdates";
import AppleOSTargetForm from "../AppleOSTargetForm";

const baseClass = "platform-tabs";

interface IPlatformTabsProps {
  currentTeamId: number;
  defaultMacOSVersion: string;
  defaultMacOSDeadline: string;
  defaultMacOSDeadlineDays: string;
  defaultMacOSUpdateNewHosts: boolean;
  defaultIOSVersion: string;
  defaultIOSDeadline: string;
  defaultIOSDeadlineDays: string;
  defaultIPadOSVersion: string;
  defaultIPadOSDeadline: string;
  defaultIPadOSDeadlineDays: string;
  defaultWindowsDeadlineDays: string;
  defaultWindowsGracePeriodDays: string;
  selectedPlatform: OSUpdatesTargetPlatform;
  onSelectPlatform: (platform: OSUpdatesTargetPlatform) => void;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
  isAppleMdmEnabled: boolean;
  isWindowsMdmEnabled: boolean;
  isAndroidMdmEnabled: boolean;
}

const PlatformTabs = ({
  currentTeamId,
  defaultMacOSDeadline,
  defaultMacOSDeadlineDays,
  defaultMacOSVersion,
  defaultMacOSUpdateNewHosts,
  defaultIOSDeadline,
  defaultIOSDeadlineDays,
  defaultIOSVersion,
  defaultIPadOSDeadline,
  defaultIPadOSDeadlineDays,
  defaultIPadOSVersion,
  defaultWindowsDeadlineDays,
  defaultWindowsGracePeriodDays,
  selectedPlatform,
  onSelectPlatform,
  refetchAppConfig,
  refetchTeamConfig,
  isAppleMdmEnabled,
  isWindowsMdmEnabled,
  isAndroidMdmEnabled,
}: IPlatformTabsProps) => {
  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while a form is
  // submitting.

  const platformByIndex: OSUpdatesTargetPlatform[] = [
    "darwin",
    "windows",
    "ios",
    "ipados",
  ];

  if (isAndroidMdmEnabled) {
    platformByIndex.push("android");
  }

  const onTabChange = (index: number) => {
    onSelectPlatform(platformByIndex[index]);
  };

  // A platform is considered "configured" when it has a minimum version
  // (Apple) or deadline days (Windows) set.
  const isMacOSConfigured = !!defaultMacOSVersion;
  const isWindowsConfigured = !!defaultWindowsDeadlineDays;
  const isIOSConfigured = !!defaultIOSVersion;
  const isIPadOSConfigured = !!defaultIPadOSVersion;

  const appleMdmEmptyState = (platformName: string) => (
    <EmptyState
      header="Turn on MDM to enforce OS updates"
      info={
        <>
          You must turn on Apple MDM to enforce OS updates for {platformName}{" "}
          hosts.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/turn-on-apple-mdm`}
            text="Learn more"
            newTab
          />
        </>
      }
      variant="form"
    />
  );

  const windowsMdmEmptyState = (
    <EmptyState
      header="Turn on MDM to enforce OS updates"
      info={
        <>
          You must turn on Windows MDM to enforce OS updates for Windows hosts.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-windows-mdm`}
            text="Learn more"
            newTab
          />
        </>
      }
      variant="form"
    />
  );

  return (
    <div className={baseClass}>
      <TabNav secondary>
        <Tabs
          defaultIndex={platformByIndex.indexOf(selectedPlatform)}
          onSelect={onTabChange}
        >
          <TabList>
            <Tab key="macOS" data-text="macOS">
              <TabText showCheck={isMacOSConfigured}>macOS</TabText>
            </Tab>
            <Tab key="Windows" data-text="Windows">
              <TabText showCheck={isWindowsConfigured}>Windows</TabText>
            </Tab>
            <Tab key="iOS" data-text="iOS">
              <TabText showCheck={isIOSConfigured}>iOS</TabText>
            </Tab>
            <Tab key="iPadOS" data-text="iPadOS">
              <TabText showCheck={isIPadOSConfigured}>iPadOS</TabText>
            </Tab>
            {isAndroidMdmEnabled && (
              <Tab key="Android" data-text="Android">
                Android
              </Tab>
            )}
          </TabList>
          <TabPanel
            className={`${baseClass}__tab-panel${
              isAppleMdmEnabled ? "" : "--empty"
            }`}
          >
            {isAppleMdmEnabled ? (
              <>
                <AppleOSTargetForm
                  currentTeamId={currentTeamId}
                  applePlatform="darwin"
                  defaultMinOsVersion={defaultMacOSVersion}
                  defaultDeadline={defaultMacOSDeadline}
                  defaultDeadlineDays={defaultMacOSDeadlineDays}
                  defaultUpdateNewHosts={defaultMacOSUpdateNewHosts}
                  key={currentTeamId}
                  refetchAppConfig={refetchAppConfig}
                  refetchTeamConfig={refetchTeamConfig}
                />
                <div className={`${baseClass}__nudge-preview`}>
                  <EndUserOSRequirementPreview platform="darwin" />
                </div>
              </>
            ) : (
              appleMdmEmptyState("macOS")
            )}
          </TabPanel>
          <TabPanel
            className={`${baseClass}__tab-panel${
              isWindowsMdmEnabled ? "" : "--empty"
            }`}
          >
            {isWindowsMdmEnabled ? (
              <>
                <WindowsTargetForm
                  currentTeamId={currentTeamId}
                  defaultDeadlineDays={defaultWindowsDeadlineDays}
                  defaultGracePeriodDays={defaultWindowsGracePeriodDays}
                  key={currentTeamId}
                  refetchAppConfig={refetchAppConfig}
                  refetchTeamConfig={refetchTeamConfig}
                />
                <div className={`${baseClass}__nudge-preview`}>
                  <EndUserOSRequirementPreview platform="windows" />
                </div>
              </>
            ) : (
              windowsMdmEmptyState
            )}
          </TabPanel>
          <TabPanel
            className={`${baseClass}__tab-panel${
              isAppleMdmEnabled ? "" : "--empty"
            }`}
          >
            {isAppleMdmEnabled ? (
              <>
                <AppleOSTargetForm
                  currentTeamId={currentTeamId}
                  applePlatform="ios"
                  defaultMinOsVersion={defaultIOSVersion}
                  defaultDeadline={defaultIOSDeadline}
                  defaultDeadlineDays={defaultIOSDeadlineDays}
                  key={currentTeamId}
                  refetchAppConfig={refetchAppConfig}
                  refetchTeamConfig={refetchTeamConfig}
                />
                <div className={`${baseClass}__nudge-preview`}>
                  <EndUserOSRequirementPreview platform="ios" />
                </div>
              </>
            ) : (
              appleMdmEmptyState("iOS")
            )}
          </TabPanel>
          <TabPanel className={`${baseClass}__tab-panel`}>
            {isAppleMdmEnabled ? (
              <>
                <AppleOSTargetForm
                  currentTeamId={currentTeamId}
                  applePlatform="ipados"
                  defaultMinOsVersion={defaultIPadOSVersion}
                  defaultDeadline={defaultIPadOSDeadline}
                  defaultDeadlineDays={defaultIPadOSDeadlineDays}
                  key={currentTeamId}
                  refetchAppConfig={refetchAppConfig}
                  refetchTeamConfig={refetchTeamConfig}
                />
                <div className={`${baseClass}__nudge-preview`}>
                  <EndUserOSRequirementPreview platform="ipados" />
                </div>
              </>
            ) : (
              appleMdmEmptyState("iPadOS")
            )}
          </TabPanel>
          {isAndroidMdmEnabled && (
            <TabPanel className={`${baseClass}__tab-panel`}>
              <div className={`${baseClass}__coming-soon`}>
                <p>
                  <b>Android updates are coming soon.</b>
                </p>
                <p>
                  Need to encourage installation of Android updates?{" "}
                  <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
                </p>
              </div>
            </TabPanel>
          )}
        </Tabs>
      </TabNav>
    </div>
  );
};

export default PlatformTabs;
