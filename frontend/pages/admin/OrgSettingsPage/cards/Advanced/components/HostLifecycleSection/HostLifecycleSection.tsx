import React from "react";
import SettingsSection from "pages/admin/components/SettingsSection";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";

import type { IAdvancedSectionProps } from "../../Advanced";

const HostLifecycleSection = ({
  isPremiumTier = false,
  onInputChange,
  formData,
  formErrors = {},
}: IAdvancedSectionProps) => {
  const {
    enableHostExpiry,
    hostExpiryWindow,
    requireHardwareAttestation,
  } = formData;

  return (
    <SettingsSection title="Host lifecycle">
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={onInputChange}
            name="enableHostExpiry"
            value={enableHostExpiry}
            parseTarget
            labelTooltipContent={
              !disableChildren && (
                <>
                  When enabled, allows automatic cleanup of
                  <br />
                  hosts that have not communicated with Fleet
                  <br />
                  in the number of days specified.{" "}
                  <em>
                    (Default: <strong>Off</strong>)
                  </em>
                </>
              )
            }
          >
            Host expiry
          </Checkbox>
        )}
      />
      {enableHostExpiry && (
        <GitOpsModeTooltipWrapper
          position="left"
          isInputField
          renderChildren={(disableChildren) => (
            <InputField
              disabled={disableChildren}
              label="Host expiry window"
              type="number"
              onChange={onInputChange}
              name="hostExpiryWindow"
              value={hostExpiryWindow}
              parseTarget
              error={formErrors.hostExpiryWindow}
            />
          )}
        />
      )}
      {isPremiumTier && (
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Checkbox
              disabled={disableChildren}
              onChange={onInputChange}
              name="requireHardwareAttestation"
              value={requireHardwareAttestation}
              parseTarget
              helpText="Enabling this setting will require macOS hosts with Apple Silicon that automatically enroll (DEP) to use ACME with Managed Device Attestation"
            >
              Require hardware attestation
            </Checkbox>
          )}
        />
      )}
    </SettingsSection>
  );
};

export default HostLifecycleSection;
