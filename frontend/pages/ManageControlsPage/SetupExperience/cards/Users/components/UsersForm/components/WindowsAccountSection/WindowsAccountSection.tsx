import React from "react";

import PATHS from "router/paths";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "windows-account-section";

interface IWindowsAccountSectionProps {
  enableManagedLocalAccount: boolean;
  onEnableManagedLocalAccountChange: (value: boolean) => void;
  isWindowsMdmEnabledAndConfigured: boolean;
}

/** Windows tab of the Users card. Windows has no end user account type to choose
 * from, so that part of the section is helper text only; the managed account
 * checkbox mirrors the macOS tab. */
const WindowsAccountSection = ({
  enableManagedLocalAccount,
  onEnableManagedLocalAccountChange,
  isWindowsMdmEnabledAndConfigured,
}: IWindowsAccountSectionProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__field-group`}>
        <h3 className={`${baseClass}__sub-header`}>End user account</h3>
        <p className={`${baseClass}__end-user-help-text`}>
          End users get the default role for the host&apos;s platform.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/end-user-accounts`}
            text="Learn more"
            newTab
          />
        </p>

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
              <Checkbox
                className={`${baseClass}__managed-local-account`}
                disabled={gitopsEnabled || !isWindowsMdmEnabledAndConfigured}
                value={enableManagedLocalAccount}
                onChange={onEnableManagedLocalAccountChange}
                helpText="A hidden local admin for remote troubleshooting."
              >
                <TooltipWrapper
                  tipContent={
                    <>
                      Creates a hidden managed local admin account for
                      <br />
                      remote troubleshooting on Windows hosts.
                    </>
                  }
                >
                  Create hidden admin
                </TooltipWrapper>
              </Checkbox>
            )}
          />
        </TooltipWrapper>
      </div>
    </div>
  );
};

export default WindowsAccountSection;
