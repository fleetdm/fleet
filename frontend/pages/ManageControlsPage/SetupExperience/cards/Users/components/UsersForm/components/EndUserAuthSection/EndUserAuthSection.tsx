import React from "react";

import PATHS from "router/paths";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import SettingsSection from "pages/admin/components/SettingsSection";

import { IUsersFormSectionProps } from "../../UsersForm";

const baseClass = "users-form";

const EndUserAuthSection = ({
  formData,
  onInputChange,
  isIdPConfigured,
  isMacMdmEnabledAndConfigured,
  gitOpsModeEnabled,
}: IUsersFormSectionProps) => {
  const { endUserAuthEnabled, lockEndUserInfo } = formData;

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
          onChange={(value: boolean) =>
            onInputChange({ name: "endUserAuthEnabled", value })
          }
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
              onChange={(value: boolean) =>
                onInputChange({ name: "lockEndUserInfo", value })
              }
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
