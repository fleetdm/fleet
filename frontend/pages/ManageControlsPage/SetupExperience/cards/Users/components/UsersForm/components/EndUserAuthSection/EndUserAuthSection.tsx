import React from "react";

import PATHS from "router/paths";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import SettingsSection from "pages/admin/components/SettingsSection";

const baseClass = "users-form";

interface IEndUserAuthSectionProps {
  endUserAuthEnabled: boolean;
  lockEndUserInfo: boolean;
  onEndUserAuthChange: (value: boolean) => void;
  onLockEndUserInfoChange: (value: boolean) => void;
  isIdPConfigured: boolean;
  isMacMdmEnabledAndConfigured: boolean;
  gitOpsModeEnabled: boolean;
}

const EndUserAuthSection = ({
  endUserAuthEnabled,
  lockEndUserInfo,
  onEndUserAuthChange,
  onLockEndUserInfoChange,
  isIdPConfigured,
  isMacMdmEnabledAndConfigured,
  gitOpsModeEnabled,
}: IEndUserAuthSectionProps) => {
  return (
    <SettingsSection title="End user authentication">
      <TooltipWrapper
        tipContent={
          !isIdPConfigured ? (
            <span>
              To enable, first connect Fleet to
              <br />
              your{" "}
              <CustomLink
                url={PATHS.ADMIN_INTEGRATIONS_SSO_END_USERS}
                text="identity provider (IdP)"
                variant="tooltip-link"
              />
              .
            </span>
          ) : undefined
        }
        disableTooltip={isIdPConfigured}
        underline={false}
        position="left"
        showArrow
      >
        <Checkbox
          disabled={gitOpsModeEnabled || !isIdPConfigured}
          value={endUserAuthEnabled}
          onChange={onEndUserAuthChange}
          helpText={
            <span>
              End users are required to authenticate with your{" "}
              <CustomLink
                url={PATHS.ADMIN_INTEGRATIONS_SSO_END_USERS}
                text="identity provider (IdP)"
              />{" "}
              when setting up new hosts. Supported for Apple (macOS, iOS,
              iPadOS),
              <br />
              Windows, Linux, and Android hosts.
            </span>
          }
        >
          Require IdP authentication
        </Checkbox>
      </TooltipWrapper>
      {endUserAuthEnabled && (
        <div className={`${baseClass}__advanced-options`}>
          <TooltipWrapper
            tipContent={
              !isMacMdmEnabledAndConfigured ? (
                <span>
                  To enable, first turn on{" "}
                  <CustomLink
                    url={PATHS.ADMIN_INTEGRATIONS_MDM_APPLE}
                    text="Apple MDM"
                    variant="tooltip-link"
                  />
                  .
                </span>
              ) : undefined
            }
            disableTooltip={!!isMacMdmEnabledAndConfigured}
            underline={false}
            position="left"
            showArrow
          >
            <Checkbox
              disabled={
                gitOpsModeEnabled ||
                !isIdPConfigured ||
                !isMacMdmEnabledAndConfigured
              }
              value={lockEndUserInfo}
              onChange={onLockEndUserInfoChange}
              helpText={
                <span>
                  Prevents macOS users from editing{" "}
                  <strong>Account Name</strong> and <strong>Full name</strong>
                  in Setup Assistant. These fields will be locked to IdP values.
                </span>
              }
            >
              Lock end user info
            </Checkbox>
          </TooltipWrapper>
        </div>
      )}
    </SettingsSection>
  );
};

export default EndUserAuthSection;
