import React from "react";
import {
  Accordion,
  AccordionItem,
  AccordionItemButton,
  AccordionItemHeading,
  AccordionItemPanel,
} from "react-accessible-accordion";

import Icon from "components/Icon";

import MacOSTargetForm from "../MacOSTargetForm";
import WindowsTargetForm from "../WindowsTargetForm";

const baseClass = "platforms-accordion";

interface IPlatformsAccordionProps {
  currentTeamId: number;
  defaultMacOSVersion: string;
  defaultMacOSDeadline: string;
  defaultWindowsDeadlineDays: string;
  defaultWindowsGracePeriodDays: string;
  onSelectAccordionItem: (platform: "mac" | "windows") => void;
}

const PlatformsAccordion = ({
  currentTeamId,
  defaultMacOSDeadline,
  defaultMacOSVersion,
  defaultWindowsDeadlineDays,
  defaultWindowsGracePeriodDays,
  onSelectAccordionItem,
}: IPlatformsAccordionProps) => {
  return (
    <Accordion
      className={`${baseClass}__accordion`}
      preExpanded={["mac"]}
      onChange={(selected) =>
        onSelectAccordionItem(selected[0] as "mac" | "windows")
      }
    >
      <AccordionItem uuid="mac">
        <AccordionItemHeading>
          <AccordionItemButton className={`${baseClass}__accordion-button`}>
            <span>macOS</span>
            <Icon name="chevron" direction="up" />
          </AccordionItemButton>
        </AccordionItemHeading>
        <AccordionItemPanel className={`${baseClass}__accordion-panel`}>
          <MacOSTargetForm
            currentTeamId={currentTeamId}
            defaultMinOsVersion={defaultMacOSVersion}
            defaultDeadline={defaultMacOSDeadline}
            key={currentTeamId}
            inAccordion
          />
        </AccordionItemPanel>
      </AccordionItem>
      <AccordionItem uuid="windows">
        <AccordionItemHeading>
          <AccordionItemButton className={`${baseClass}__accordion-button`}>
            <span>Windows</span>
            <Icon name="chevron" direction="up" />
          </AccordionItemButton>
        </AccordionItemHeading>
        <AccordionItemPanel className={`${baseClass}__accordion-panel`}>
          <WindowsTargetForm
            currentTeamId={currentTeamId}
            defaultDeadlineDays={defaultWindowsDeadlineDays}
            defaultGracePeriodDays={defaultWindowsGracePeriodDays}
            key={currentTeamId}
            inAccordion
          />
        </AccordionItemPanel>
      </AccordionItem>
    </Accordion>
  );
};

export default PlatformsAccordion;
