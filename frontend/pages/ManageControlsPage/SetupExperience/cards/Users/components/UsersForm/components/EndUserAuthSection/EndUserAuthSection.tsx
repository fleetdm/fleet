import React from "react";

import PATHS from "router/paths";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import SettingsSection from "pages/admin/components/SettingsSection";

import TurnOnMdmTooltipWrapper from "../TurnOnMdmTooltipWrapper";

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
            <>
              To enable, first connect Fleet to your{" "}
              <CustomLink
                url={PATHS.ADMIN_INTEGRATIONS_SSO_END_USERS}
                text="identity provider (IdP)"
                variant="tooltip-link"
              />
              .
            </>
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
              />
              . ChromeOS not supported.
            </span>
          }
        >
          Require IdP authentication
        </Checkbox>
      </TooltipWrapper>
      {endUserAuthEnabled && (
        <div className={`${baseClass}__advanced-options`}>
          <TurnOnMdmTooltipWrapper
            platform="apple"
            isMdmEnabledAndConfigured={!!isMacMdmEnabledAndConfigured}
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
                  <strong>Account Name</strong> and <strong>Full name</strong>{" "}
                  will be locked to IdP values in Setup Assistant. macOS only.
                </span>
              }
            >
              Lock end user info
            </Checkbox>
          </TurnOnMdmTooltipWrapper>
        </div>
      )}
    </SettingsSection>
  );
};

export default EndUserAuthSection;
