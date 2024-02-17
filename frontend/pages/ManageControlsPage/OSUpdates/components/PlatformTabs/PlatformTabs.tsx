import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import TabsWrapper from "components/TabsWrapper";

import MacOSTargetForm from "../MacOSTargetForm";
import WindowsTargetForm from "../WindowsTargetForm";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

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
            <Tab>macOS</Tab>
            <Tab>Windows</Tab>
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
        </Tabs>
      </TabsWrapper>
    </div>
  );
};

export default PlatformTabs;
