import React from "react";
import SettingsSection from "pages/admin/components/SettingsSection";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import type { IAdvancedSectionProps } from "../../Advanced";

const baseClass = "host-lifecycle-section";

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
      <div className={`${baseClass}__host-expiry-window`}>
        <GitOpsModeTooltipWrapper
          position="left"
          isInputField
          renderChildren={(disableChildren) => {
            const hostExpiryWindowField = (
              <InputField
                disabled={!enableHostExpiry || disableChildren}
                label="Host expiry window"
                type="number"
                onChange={onInputChange}
                name="hostExpiryWindow"
                value={hostExpiryWindow}
                parseTarget
                error={formErrors.hostExpiryWindow}
              />
            );

            return !enableHostExpiry && !disableChildren ? (
              <TooltipWrapper
                className={`${baseClass}__disabled-tooltip`}
                tipContent="Enable host expiry to edit this setting."
                position="top"
                underline={false}
                showArrow
              >
                {hostExpiryWindowField}
              </TooltipWrapper>
            ) : (
              hostExpiryWindowField
            );
          }}
        />
      </div>
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
