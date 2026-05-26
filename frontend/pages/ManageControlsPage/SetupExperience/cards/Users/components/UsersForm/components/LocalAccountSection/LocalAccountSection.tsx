import React from "react";

import PATHS from "router/paths";

import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import Radio from "components/forms/fields/Radio";
import SettingsSection from "pages/admin/components/SettingsSection";
import { EndUserLocalAccountType } from "services/entities/mdm";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { IUsersFormData, IUsersFormSectionProps } from "../../UsersForm";

const baseClass = "local-account-section";

// Standard and None require a managed local admin account; only Admin leaves
// the checkbox free. Used for both the in-form display state and the value
// sent in the save payload.
export const effectiveEnableManagedLocalAccount = (formData: IUsersFormData) =>
  formData.enableManagedLocalAccount ||
  formData.localAccountType !== EndUserLocalAccountType.Admin;

const LocalAccountSection = ({
  formData,
  onInputChange,
  isMacMdmEnabledAndConfigured,
}: IUsersFormSectionProps) => {
  const { localAccountType } = formData;
  const forcedByLocalAccountType =
    localAccountType !== EndUserLocalAccountType.Admin;

  return (
    <SettingsSection
      title="Local account"
      subTitle={
        <span>
          Currently supported for macOS hosts. End users get the default role
          for all other platforms.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/end-user-accounts`}
            text="Learn more"
            newTab
          />
        </span>
      }
      className={baseClass}
    >
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
        <GitOpsModeTooltipWrapper
          position="left"
          tipOffset={8}
          renderChildren={(gitopsEnabled) => {
            return (
              <div className={`${baseClass}__field-group`}>
                <fieldset className="form-field">
                  <Radio
                    name="localAccountType"
                    id="localAccountTypeAdmin"
                    label="Admin"
                    helpText="End user can add and manage other users, install apps, and change settings."
                    value={EndUserLocalAccountType.Admin}
                    disabled={gitopsEnabled || !isMacMdmEnabledAndConfigured}
                    checked={localAccountType === EndUserLocalAccountType.Admin}
                    onChange={(val) => {
                      onInputChange({ name: "localAccountType", value: val });
                    }}
                  />
                  <Radio
                    name="localAccountType"
                    id="localAccountTypeStandard"
                    label="Standard"
                    helpText={
                      <span>
                        End user can install apps and change their own settings,
                        but can&apos;t add other users or change other
                        users&apos; settings.
                      </span>
                    }
                    value={EndUserLocalAccountType.Standard}
                    checked={
                      localAccountType === EndUserLocalAccountType.Standard
                    }
                    disabled={gitopsEnabled || !isMacMdmEnabledAndConfigured}
                    onChange={(val) => {
                      onInputChange({ name: "localAccountType", value: val });
                    }}
                  />
                  <Radio
                    name="localAccountType"
                    id="localAccountTypeNone"
                    label="Skip (no account)"
                    helpText="No user account will be created during Setup Assistant and authentication must be handled by an IdP or other workflow."
                    disabled={gitopsEnabled || !isMacMdmEnabledAndConfigured}
                    value={EndUserLocalAccountType.None}
                    checked={localAccountType === EndUserLocalAccountType.None}
                    onChange={(val) => {
                      onInputChange({ name: "localAccountType", value: val });
                    }}
                  />
                </fieldset>
                <Checkbox
                  className={`${baseClass}__managed-local-account`}
                  disabled={
                    gitopsEnabled ||
                    !isMacMdmEnabledAndConfigured ||
                    forcedByLocalAccountType
                  }
                  iconTooltipContent={
                    forcedByLocalAccountType ? (
                      <span>
                        There must be at least one admin account on the host.
                      </span>
                    ) : undefined
                  }
                  value={effectiveEnableManagedLocalAccount(formData)}
                  onChange={(value: boolean) =>
                    onInputChange({ name: "enableManagedLocalAccount", value })
                  }
                  helpText={
                    <span>
                      Fleet creates a user (_fleetadmin) and unique password for
                      each macOS host, accessible in <b>Host details</b> &gt;{" "}
                      <b>Show managed account</b>.
                    </span>
                  }
                >
                  <TooltipWrapper
                    tipContent={
                      <>
                        Creates a hidden managed local admin account for
                        <br />
                        remote troubleshooting on macOS hosts.
                      </>
                    }
                  >
                    Create hidden admin
                  </TooltipWrapper>
                </Checkbox>
              </div>
            );
          }}
        />
      </TooltipWrapper>
    </SettingsSection>
  );
};

export default LocalAccountSection;
