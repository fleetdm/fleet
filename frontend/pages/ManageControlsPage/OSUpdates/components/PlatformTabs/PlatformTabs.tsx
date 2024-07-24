import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import TabsWrapper from "components/TabsWrapper";

import WindowsTargetForm from "../WindowsTargetForm";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";
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
  selectedPlatform: OSUpdatesSupportedPlatform;
  onSelectPlatform: (platform: OSUpdatesSupportedPlatform) => void;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
  isWindowsMdmEnabled: boolean;
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
}: IPlatformTabsProps) => {
  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while a form is
  // submitting.

  const PLATFORM_BY_INDEX: OSUpdatesSupportedPlatform[] = isWindowsMdmEnabled
    ? ["darwin", "windows", "ios", "ipados"]
    : ["darwin", "ios", "ipados"];

  const onTabChange = (index: number) => {
    onSelectPlatform(PLATFORM_BY_INDEX[index]);
  };

  return (
    <div className={baseClass}>
      <TabsWrapper>
        <Tabs
          defaultIndex={PLATFORM_BY_INDEX.indexOf(selectedPlatform)}
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
        </Tabs>
      </TabsWrapper>
    </div>
  );
};

export default PlatformTabs;
