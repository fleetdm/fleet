import React from "react";

import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import ManagedAccountCheckbox from "../ManagedAccountCheckbox";
import TurnOnMdmTooltipWrapper from "../TurnOnMdmTooltipWrapper";

const baseClass = "windows-account-section";

interface IWindowsAccountSectionProps {
  enableManagedLocalAccount: boolean;
  onEnableManagedLocalAccountChange: (value: boolean) => void;
  isWindowsMdmEnabledAndConfigured: boolean;
}

/** Windows tab of the Users card. The managed account checkbox mirrors the macOS tab, including the disabled state that points admins at Windows MDM when it isn't turned on yet. */
const WindowsAccountSection = ({
  enableManagedLocalAccount,
  onEnableManagedLocalAccountChange,
  isWindowsMdmEnabledAndConfigured,
}: IWindowsAccountSectionProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__field-group`}>
        <h3 className={`${baseClass}__sub-header`}>End user account</h3>
        <div className={`${baseClass}__end-user-help-text`}>
          End users get the default role for the host&apos;s platform.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/end-user-accounts`}
            text="Learn more"
            newTab
          />
        </div>

        <h3 className={`${baseClass}__sub-header`}>Managed account</h3>
        <TurnOnMdmTooltipWrapper
          platform="windows"
          isMdmEnabledAndConfigured={isWindowsMdmEnabledAndConfigured}
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
        </TurnOnMdmTooltipWrapper>
      </div>
    </div>
  );
};

export default WindowsAccountSection;
