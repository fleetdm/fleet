import React from "react";

import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import PATHS from "router/paths";

const MDM_BY_PLATFORM = {
  apple: { url: PATHS.ADMIN_INTEGRATIONS_MDM_APPLE, text: "Apple MDM" },
  windows: { url: PATHS.ADMIN_INTEGRATIONS_MDM_WINDOWS, text: "Windows MDM" },
} as const;

// Pin the tooltip to the first radio or checkbox row, not the middle of the
// whole group (help text included).
export const CONTROL_ROW_SELECTOR = "[role='checkbox'], .radio__radio-control";

interface ITurnOnMdmTooltipWrapperProps {
  platform: keyof typeof MDM_BY_PLATFORM;
  isMdmEnabledAndConfigured: boolean;
  children: React.ReactNode;
}

/** Explains that a control is disabled because the platform's MDM isn't turned on yet, and links to the page that turns it on. */
const TurnOnMdmTooltipWrapper = ({
  platform,
  isMdmEnabledAndConfigured,
  children,
}: ITurnOnMdmTooltipWrapperProps) => {
  const { url, text } = MDM_BY_PLATFORM[platform];

  return (
    <TooltipWrapper
      tipContent={
        !isMdmEnabledAndConfigured ? (
          <span>
            To enable, first turn on{" "}
            <CustomLink url={url} text={text} variant="tooltip-link" />.
          </span>
        ) : undefined
      }
      disableTooltip={isMdmEnabledAndConfigured}
      underline={false}
      position="left"
      tipOffset={8}
      showArrow
      anchorChildSelector={CONTROL_ROW_SELECTOR}
    >
      {children}
    </TooltipWrapper>
  );
};

export default TurnOnMdmTooltipWrapper;
