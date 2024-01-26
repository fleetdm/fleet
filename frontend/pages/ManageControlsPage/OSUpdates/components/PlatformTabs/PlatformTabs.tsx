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
  onSelectAccordionItem: (platform: OSUpdatesSupportedPlatform) => void;
}

const PlatformTabs = ({
  currentTeamId,
  defaultMacOSDeadline,
  defaultMacOSVersion,
  defaultWindowsDeadlineDays,
  defaultWindowsGracePeriodDays,
  onSelectAccordionItem,
}: IPlatformTabsProps) => {
  return (
    <div className={baseClass}>
      <TabsWrapper>
        <Tabs
          onSelect={(currentIndex) =>
            onSelectAccordionItem(currentIndex === 0 ? "darwin" : "windows")
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
              inAccordion
            />
          </TabPanel>
          <TabPanel>
            <WindowsTargetForm
              currentTeamId={currentTeamId}
              defaultDeadlineDays={defaultWindowsDeadlineDays}
              defaultGracePeriodDays={defaultWindowsGracePeriodDays}
              key={currentTeamId}
              inAccordion
            />
          </TabPanel>
        </Tabs>
      </TabsWrapper>
    </div>
  );
};

export default PlatformTabs;
