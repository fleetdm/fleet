import React from "react";

import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import SettingsSection from "pages/admin/components/SettingsSection";
import PATHS from "router/paths";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import type { IAdvancedSectionProps } from "../../Advanced";

const baseClass = "host-lifecycle-section";

const HostLifecycleSection = ({
  isPremiumTier = false,
  onInputChange,
  formData,
  formErrors = {},
  appConfig,
}: IAdvancedSectionProps) => {
  const {
    enableHostExpiry,
    hostExpiryWindow,
    requireHardwareAttestation,
    onlyAllowAppleBusinessEnrollment,
  } = formData;

  const isAppleBusinessSetUp =
    !!appConfig?.mdm.enabled_and_configured &&
    !!appConfig?.mdm.apple_business_enabled_and_configured;

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
          renderChildren={(disableChildren) => (
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
          )}
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
              <TooltipWrapper
                tipContent={
                  <span>
                    Turn on{" "}
                    <CustomLink
                      url={PATHS.ADMIN_INTEGRATIONS_MDM_APPLE}
                      text="Apple MDM"
                      variant="tooltip-link"
                    />{" "}
                    and connect to{" "}
                    <CustomLink
                      url={PATHS.ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER}
                      text="Apple Business"
                      variant="tooltip-link"
                    />{" "}
                    to use this feature.
                  </span>
                }
                disableTooltip={disableChildren || isAppleBusinessSetUp}
                position="top"
                showArrow
                underline={false}
              >
                <Checkbox
                  disabled={disableChildren || !isAppleBusinessSetUp}
                  onChange={onInputChange}
                  name="onlyAllowAppleBusinessEnrollment"
                  value={onlyAllowAppleBusinessEnrollment}
                  parseTarget
                  helpText="Only allow Apple (macOS, iOS, iPadOS) hosts to enroll via Apple Business."
                >
                  Allow only Apple Business enrollments
                </Checkbox>
              </TooltipWrapper>
            )}
          />
        </>
      )}
    </SettingsSection>
  );
};

export default HostLifecycleSection;
