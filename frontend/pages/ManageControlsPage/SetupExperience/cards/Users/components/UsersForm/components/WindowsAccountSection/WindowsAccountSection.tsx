import React from "react";

import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import ManagedAccountCheckbox from "../ManagedAccountCheckbox";

const baseClass = "windows-account-section";

interface IWindowsAccountSectionProps {
  enableManagedLocalAccount: boolean;
  onEnableManagedLocalAccountChange: (value: boolean) => void;
}

/** Windows tab of the Users card. The tab is only rendered when Windows MDM is configured, so unlike the macOS section there is no "turn on MDM first" state to handle here. */
const WindowsAccountSection = ({
  enableManagedLocalAccount,
  onEnableManagedLocalAccountChange,
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
        <GitOpsModeTooltipWrapper
          position="left"
          tipOffset={8}
          isInputField
          renderChildren={(gitopsEnabled) => (
            <ManagedAccountCheckbox
              disabled={!!gitopsEnabled}
              value={enableManagedLocalAccount}
              onChange={onEnableManagedLocalAccountChange}
            />
          )}
        />
      </div>
    </div>
  );
};

export default WindowsAccountSection;
