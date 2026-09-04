import React from "react";
import SettingsSection from "pages/admin/components/SettingsSection";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

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
    onlyAllowAppleBusinessEnrollment,
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
                  When enabled, allows automatic cleanup of hosts that have not
                  communicated with Fleet in the number of days specified.
                  <br />
                  <i>
                    (Default: <strong>Off</strong>)
                  </i>
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
        <>
          <GitOpsModeTooltipWrapper
            position="left"
            renderChildren={(disableChildren) => (
              <Checkbox
                disabled={disableChildren}
                onChange={onInputChange}
                name="requireHardwareAttestation"
                value={requireHardwareAttestation}
                parseTarget
                helpText={
                  <span>
                    Apple hosts that support Managed Device Attestation that
                    auto-enroll (DEP) will use ACME with Managed Device
                    Attestation.
                    <br /> If &quot;Allow only Apple Business enrollments&quot;
                    is also enabled, some hosts may be unable to enroll.{" "}
                    <CustomLink
                      text="Learn more"
                      newTab
                      url={`${LEARN_MORE_ABOUT_BASE_LINK}/device-attestation`}
                    />
                  </span>
                }
              >
                Use hardware attestation
              </Checkbox>
            )}
          />
          <GitOpsModeTooltipWrapper
            position="left"
            renderChildren={(disableChildren) => (
              <Checkbox
                disabled={disableChildren}
                onChange={onInputChange}
                name="onlyAllowAppleBusinessEnrollment"
                value={onlyAllowAppleBusinessEnrollment}
                parseTarget
                helpText="Enabling this setting will allow only hosts from Apple Business to use MDM features. Manually turning on MDM won't work."
              >
                Allow only Apple Business enrollments
              </Checkbox>
            )}
          />
        </>
      )}
    </SettingsSection>
  );
};

export default HostLifecycleSection;
