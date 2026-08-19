import React from "react";

import Radio from "components/forms/fields/Radio";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { EndUserLocalAccountType } from "interfaces/mdm";

import ManagedAccountCheckbox from "../ManagedAccountCheckbox";
import TurnOnMdmTooltipWrapper from "../TurnOnMdmTooltipWrapper";
import { IUsersFormData } from "../../UsersForm";

const baseClass = "local-account-section";

// Standard and None require a managed local admin account; only Admin leaves
// the checkbox free. Used for both the in-form display state and the value
// sent in the save payload.
export const effectiveEnableManagedLocalAccount = (formData: IUsersFormData) =>
  formData.enableManagedLocalAccount ||
  formData.localAccountType !== EndUserLocalAccountType.ADMIN;

interface ILocalAccountSectionProps {
  formData: IUsersFormData;
  onLocalAccountTypeChange: (value: EndUserLocalAccountType) => void;
  onEnableManagedLocalAccountChange: (value: boolean) => void;
  isMacMdmEnabledAndConfigured: boolean;
}

const LocalAccountSection = ({
  formData,
  onLocalAccountTypeChange,
  onEnableManagedLocalAccountChange,
  isMacMdmEnabledAndConfigured,
}: ILocalAccountSectionProps) => {
  const { localAccountType } = formData;
  const forcedByLocalAccountType =
    localAccountType !== EndUserLocalAccountType.ADMIN;
  return (
    <div className={baseClass}>
      <TurnOnMdmTooltipWrapper
        platform="apple"
        isMdmEnabledAndConfigured={isMacMdmEnabledAndConfigured}
      >
        <GitOpsModeTooltipWrapper
          position="left"
          tipOffset={8}
          isInputField
          renderChildren={(gitopsEnabled) => {
            return (
              <div className={`${baseClass}__field-group`}>
                <h3 className={`${baseClass}__sub-header`}>End user account</h3>
                <fieldset className="form-field">
                  <Radio
                    name="localAccountType"
                    id="localAccountTypeAdmin"
                    label="Admin"
                    helpText="End user can add and manage other users, install apps, and change settings."
                    value={EndUserLocalAccountType.ADMIN}
                    disabled={gitopsEnabled || !isMacMdmEnabledAndConfigured}
                    checked={localAccountType === EndUserLocalAccountType.ADMIN}
                    onChange={(val) =>
                      onLocalAccountTypeChange(val as EndUserLocalAccountType)
                    }
                  />
                  <Radio
                    name="localAccountType"
                    id="localAccountTypeStandard"
                    label="Standard"
                    helpText="End user can install apps and change their own settings only."
                    value={EndUserLocalAccountType.STANDARD}
                    checked={
                      localAccountType === EndUserLocalAccountType.STANDARD
                    }
                    disabled={gitopsEnabled || !isMacMdmEnabledAndConfigured}
                    onChange={(val) =>
                      onLocalAccountTypeChange(val as EndUserLocalAccountType)
                    }
                  />
                  <Radio
                    name="localAccountType"
                    id="localAccountTypeNone"
                    label="Skip (no account)"
                    helpText="No user account will be created and authentication must be handled by an IdP or other workflow."
                    disabled={gitopsEnabled || !isMacMdmEnabledAndConfigured}
                    value={EndUserLocalAccountType.NONE}
                    checked={localAccountType === EndUserLocalAccountType.NONE}
                    onChange={(val) =>
                      onLocalAccountTypeChange(val as EndUserLocalAccountType)
                    }
                  />
                </fieldset>

                <h3 className={`${baseClass}__sub-header`}>Managed account</h3>
                <ManagedAccountCheckbox
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
                  onChange={onEnableManagedLocalAccountChange}
                />
              </div>
            );
          }}
        />
      </TurnOnMdmTooltipWrapper>
    </div>
  );
};

export default LocalAccountSection;
