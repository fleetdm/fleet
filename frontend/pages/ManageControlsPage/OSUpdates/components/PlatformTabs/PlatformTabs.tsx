import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import TabsWrapper from "components/TabsWrapper";
import CustomLink from "components/CustomLink";
import { SUPPORT_LINK } from "utilities/constants";

import WindowsTargetForm from "../WindowsTargetForm";
import { OSUpdatesTargetPlatform } from "../../OSUpdates";
import AppleOSTargetForm from "../AppleOSTargetForm";

const baseClass = "platform-tabs";

interface IPlatformTabsProps {
  currentTeamId: number;
  defaultMacOSVersion: string;
  defaultMacOSDeadline: string;
  defaultIOSVersion: string;
  defaultIOSDeadline: string;
  defaultIPadOSVersion: string;
  defaultIPadOSDeadline: string;
  defaultWindowsDeadlineDays: string;
  defaultWindowsGracePeriodDays: string;
  selectedPlatform: OSUpdatesTargetPlatform;
  onSelectPlatform: (platform: OSUpdatesTargetPlatform) => void;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
  isWindowsMdmEnabled: boolean;
  isAndroidMdmEnabled: boolean;
}

const PlatformTabs = ({
  currentTeamId,
  defaultMacOSDeadline,
  defaultMacOSVersion,
  defaultIOSDeadline,
  defaultIOSVersion,
  defaultIPadOSDeadline,
  defaultIPadOSVersion,
  defaultWindowsDeadlineDays,
  defaultWindowsGracePeriodDays,
  selectedPlatform,
  onSelectPlatform,
  refetchAppConfig,
  refetchTeamConfig,
  isWindowsMdmEnabled,
  isAndroidMdmEnabled,
}: IPlatformTabsProps) => {
  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while a form is
  // submitting.

  const platformByIndex: OSUpdatesTargetPlatform[] = isWindowsMdmEnabled
    ? ["darwin", "windows", "ios", "ipados"]
    : ["darwin", "ios", "ipados"];

  if (isAndroidMdmEnabled) {
    platformByIndex.push("android");
  }

  const onTabChange = (index: number) => {
    onSelectPlatform(platformByIndex[index]);
  };

  return (
    <div className={baseClass}>
      <TabsWrapper>
        <Tabs
          defaultIndex={platformByIndex.indexOf(selectedPlatform)}
          onSelect={onTabChange}
        >
          <TabList>
            {/* Bolding text when the tab is active causes a layout shift so
            we add a hidden pseudo element with the same text string */}
            <Tab key="macOS" data-text="macOS">
              macOS
            </Tab>
            {isWindowsMdmEnabled && (
              <Tab key="Windows" data-text="Windows">
                Windows
              </Tab>
            )}
            <Tab key="iOS" data-text="iOS">
              iOS
            </Tab>
            <Tab key="iPadOS" data-text="iPadOS">
              iPadOS
            </Tab>
            {isAndroidMdmEnabled && (
              <Tab key="Android" data-text="Android">
                Android
              </Tab>
            )}
          </TabList>
          <TabPanel>
            <AppleOSTargetForm
              currentTeamId={currentTeamId}
              applePlatform="darwin"
              defaultMinOsVersion={defaultMacOSVersion}
              defaultDeadline={defaultMacOSDeadline}
              key={currentTeamId}
              refetchAppConfig={refetchAppConfig}
              refetchTeamConfig={refetchTeamConfig}
            />
          </TabPanel>
          {isWindowsMdmEnabled && (
            <TabPanel>
              <WindowsTargetForm
                currentTeamId={currentTeamId}
                defaultDeadlineDays={defaultWindowsDeadlineDays}
                defaultGracePeriodDays={defaultWindowsGracePeriodDays}
                key={currentTeamId}
                refetchAppConfig={refetchAppConfig}
                refetchTeamConfig={refetchTeamConfig}
              />
            </TabPanel>
          )}
          <TabPanel>
            <AppleOSTargetForm
              currentTeamId={currentTeamId}
              applePlatform="ios"
              defaultMinOsVersion={defaultIOSVersion}
              defaultDeadline={defaultIOSDeadline}
              key={currentTeamId}
              refetchAppConfig={refetchAppConfig}
              refetchTeamConfig={refetchTeamConfig}
            />
          </TabPanel>
          <TabPanel>
            <AppleOSTargetForm
              currentTeamId={currentTeamId}
              applePlatform="ipados"
              defaultMinOsVersion={defaultIPadOSVersion}
              defaultDeadline={defaultIPadOSDeadline}
              key={currentTeamId}
              refetchAppConfig={refetchAppConfig}
              refetchTeamConfig={refetchTeamConfig}
            />
          </TabPanel>
          {isAndroidMdmEnabled && (
            <TabPanel>
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
      </TabsWrapper>
    </div>
  );
};

export default PlatformTabs;
