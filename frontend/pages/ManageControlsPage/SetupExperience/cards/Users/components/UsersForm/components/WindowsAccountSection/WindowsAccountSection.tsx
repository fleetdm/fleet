import React from "react";
import PATHS from "router/paths";

import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import ManagedAccountCheckbox from "../ManagedAccountCheckbox";

const baseClass = "windows-account-section";

interface IWindowsAccountSectionProps {
  enableManagedLocalAccount: boolean;
  onEnableManagedLocalAccountChange: (value: boolean) => void;
  isWindowsMdmEnabledAndConfigured: boolean;
}

/** Windows tab of the Users card. Windows has no end user account type to choose from, so that part
 * of the section is helper text only; the managed account checkbox mirrors the macOS tab, including
 * the disabled state that points admins at Windows MDM when it isn't turned on yet. */
const WindowsAccountSection = ({
  enableManagedLocalAccount,
  onEnableManagedLocalAccountChange,
  isWindowsMdmEnabledAndConfigured,
}: IWindowsAccountSectionProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__field-group`}>
        <h3 className={`${baseClass}__sub-header`}>End user account</h3>
        {/* Not a <p>: CustomLink's external-link icon renders a <div>, which a paragraph
        cannot contain, so the parser would close the paragraph early. */}
        <div className={`${baseClass}__end-user-help-text`}>
          End users get the default role for the host&apos;s platform.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/end-user-accounts`}
            text="Learn more"
            newTab
          />
        </div>

        <h3 className={`${baseClass}__sub-header`}>Managed account</h3>
        <TooltipWrapper
          tipContent={
            !isWindowsMdmEnabledAndConfigured ? (
              <span>
                To enable, first turn on{" "}
                <CustomLink
                  url={PATHS.ADMIN_INTEGRATIONS_MDM_WINDOWS}
                  text="Windows MDM"
                  variant="tooltip-link"
                />
                .
              </span>
            ) : undefined
          }
          disableTooltip={isWindowsMdmEnabledAndConfigured}
          underline={false}
          position="left"
          showArrow
        >
          <GitOpsModeTooltipWrapper
            position="left"
            tipOffset={8}
            isInputField
            renderChildren={(gitopsEnabled) => (
              <ManagedAccountCheckbox
                disabled={!!gitopsEnabled || !isWindowsMdmEnabledAndConfigured}
                value={enableManagedLocalAccount}
                onChange={onEnableManagedLocalAccountChange}
              />
            )}
          />
        </TooltipWrapper>
      </div>
    </div>
  );
};

export default WindowsAccountSection;
