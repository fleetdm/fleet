import React from "react";
import {
  Accordion,
  AccordionItem,
  AccordionItemButton,
  AccordionItemHeading,
  AccordionItemPanel,
  AccordionItemState,
} from "react-accessible-accordion";
import classnames from "classnames";

import Icon from "components/Icon";

import MacOSTargetForm from "../MacOSTargetForm";
import WindowsTargetForm from "../WindowsTargetForm";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

const baseClass = "platforms-accordion";

const generateIconClassNames = (expanded?: boolean) => {
  return classnames(`${baseClass}__item-icon`, {
    [`${baseClass}__item-closed`]: !expanded,
  });
};

interface IPlatformsAccordionProps {
  currentTeamId: number;
  defaultMacOSVersion: string;
  defaultMacOSDeadline: string;
  defaultWindowsDeadlineDays: string;
  defaultWindowsGracePeriodDays: string;
  onSelectAccordionItem: (platform: OSUpdatesSupportedPlatform) => void;
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
        onSelectAccordionItem(selected[0] as OSUpdatesSupportedPlatform)
      }
    >
      <AccordionItem uuid="mac">
        <AccordionItemHeading>
          <AccordionItemButton className={`${baseClass}__accordion-button`}>
            <span>macOS</span>
            <AccordionItemState>
              {({ expanded }) => (
                <Icon
                  name="chevron-up"
                  className={generateIconClassNames(expanded)}
                />
              )}
            </AccordionItemState>
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
            <AccordionItemState>
              {({ expanded }) => (
                <Icon
                  name="chevron-up"
                  className={generateIconClassNames(expanded)}
                />
              )}
            </AccordionItemState>
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
