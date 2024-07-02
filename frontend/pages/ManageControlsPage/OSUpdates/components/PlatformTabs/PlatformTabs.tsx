import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import TabsWrapper from "components/TabsWrapper";

import MacOSTargetForm from "../MacOSTargetForm";
import WindowsTargetForm from "../WindowsTargetForm";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";
import EmptyTargetForm from "../EmptyTargetForm";

const baseClass = "platform-tabs";

interface IPlatformTabsProps {
  currentTeamId: number;
  defaultMacOSVersion: string;
  defaultMacOSDeadline: string;
  defaultWindowsDeadlineDays: string;
  defaultWindowsGracePeriodDays: string;
  selectedPlatform: OSUpdatesSupportedPlatform;
  onSelectPlatform: (platform: OSUpdatesSupportedPlatform) => void;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
}

const PlatformTabs = ({
  currentTeamId,
  defaultMacOSDeadline,
  defaultMacOSVersion,
  defaultWindowsDeadlineDays,
  defaultWindowsGracePeriodDays,
  selectedPlatform,
  onSelectPlatform,
  refetchAppConfig,
  refetchTeamConfig,
}: IPlatformTabsProps) => {
  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while a form is
  // submitting.
  return (
    <div className={baseClass}>
      <TabsWrapper>
        <Tabs
          defaultIndex={selectedPlatform === "darwin" ? 0 : 1}
          onSelect={(currentIndex) =>
            onSelectPlatform(currentIndex === 0 ? "darwin" : "windows")
          }
        >
          <TabList>
            {/* Bolding text when the tab is active causes a layout shift so
            we add a hidden pseudo element with the same text string */}
            <Tab key={"macOS"} data-text={"macOS"}>
              macOS
            </Tab>
            <Tab key={"Windows"} data-text={"Windows"}>
              Windows
            </Tab>
            <Tab key={"iOS"} data-text={"iOS"}>
              iOS
            </Tab>
            <Tab key={"iPadOS"} data-text={"iPadOS"}>
              iPadOS
            </Tab>
          </TabList>
          <TabPanel>
            <MacOSTargetForm
              currentTeamId={currentTeamId}
              defaultMinOsVersion={defaultMacOSVersion}
              defaultDeadline={defaultMacOSDeadline}
              key={currentTeamId}
              refetchAppConfig={refetchAppConfig}
              refetchTeamConfig={refetchTeamConfig}
            />
          </TabPanel>
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
          <TabPanel>
            <EmptyTargetForm targetPlatform="iOS" />
          </TabPanel>
          <TabPanel>
            <EmptyTargetForm targetPlatform="iPadOS" />
          </TabPanel>
        </Tabs>
      </TabsWrapper>
    </div>
  );
};

export default PlatformTabs;
