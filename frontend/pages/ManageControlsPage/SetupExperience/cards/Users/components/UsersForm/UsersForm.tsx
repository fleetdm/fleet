import React, { useContext, useEffect, useState } from "react";

import { IInputFieldParseTarget } from "interfaces/form_field";
import mdmAPI, { EndUserLocalAccountType } from "services/entities/mdm";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import EndUserAuthSection from "./components/EndUserAuthSection";
import LocalAccountSection, {
  effectiveEnableManagedLocalAccount,
} from "./components/LocalAccountSection/LocalAccountSection";

const baseClass = "users-form";

export interface IUsersFormData {
  endUserAuthEnabled: boolean;
  lockEndUserInfo: boolean;
  enableManagedLocalAccount: boolean;
  localAccountType: EndUserLocalAccountType;
}

export interface IUsersFormErrors {
  endUserAuthEnabled?: string | null;
  lockEndUserInfo?: string | null;
  enableManagedLocalAccount?: string | null;
  localAccountType?: EndUserLocalAccountType;
}

export type UsersInputChangeFn = ({
  name,
  value,
}: IInputFieldParseTarget) => void;

export interface IUsersFormSectionProps {
  formData: IUsersFormData;
  formErrors: IUsersFormErrors;
  onInputChange: UsersInputChangeFn;
  onInputBlur?: () => void;
  isIdPConfigured: boolean;
  isMacMdmEnabledAndConfigured: boolean;
  gitOpsModeEnabled: boolean;
}

interface IUsersFormProps {
  currentTeamId: number;
  defaultIsEndUserAuthEnabled: boolean;
  defaultLockEndUserInfo: boolean;
  defaultEnableManagedLocalAccount: boolean;
  /** The radio value to start from. Defaults to the option that doesn't
   * force the managed local account on. */
  defaultLocalAccountType?: EndUserLocalAccountType;
  isIdPConfigured: boolean;
}

// No validations are required today. The function exists so new fields can add
// validation in one place without changing any call sites.
const validateFormData = (formData: IUsersFormData) => {
  const errors: Record<string, string> = {};
  // Reference formData so the unused-parameter warning stays quiet while there
  // are no rules to evaluate. Replace this with real checks when fields need
  // validation.
  if (!formData) return errors;
  return errors;
};

const UsersForm = ({
  currentTeamId,
  defaultIsEndUserAuthEnabled,
  defaultLockEndUserInfo,
  defaultEnableManagedLocalAccount,
  defaultLocalAccountType = EndUserLocalAccountType.Admin,
  isIdPConfigured,
}: IUsersFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config, isMacMdmEnabledAndConfigured } = useContext(AppContext);
  const gitOpsModeEnabled = !!config?.gitops.gitops_mode_enabled;

  const [formData, setFormData] = useState<IUsersFormData>({
    endUserAuthEnabled: defaultIsEndUserAuthEnabled,
    lockEndUserInfo: defaultLockEndUserInfo,
    enableManagedLocalAccount: defaultEnableManagedLocalAccount,
    localAccountType: defaultLocalAccountType,
  });
  const [formErrors, setFormErrors] = useState<IUsersFormErrors>({});
  const [isUpdating, setIsUpdating] = useState(false);

  // Re-sync local state when the parent refetches config (e.g. team switch).
  // useState initializers only run on first mount, so without this the form
  // would show and save the previous team's settings.
  useEffect(() => {
    setFormData({
      endUserAuthEnabled: defaultIsEndUserAuthEnabled,
      lockEndUserInfo: defaultLockEndUserInfo,
      enableManagedLocalAccount: defaultEnableManagedLocalAccount,
      localAccountType: defaultLocalAccountType,
    });
  }, [
    defaultIsEndUserAuthEnabled,
    defaultLockEndUserInfo,
    defaultEnableManagedLocalAccount,
    defaultLocalAccountType,
  ]);

  const onInputChange: UsersInputChangeFn = ({ name, value }) => {
    const next: IUsersFormData = { ...formData, [name]: value };
    // Sync lock end user info with EUA only when Apple MDM is configured. Without
    // Apple MDM the field is read-only, so we leave whatever value came from the
    // backend untouched.
    if (name === "endUserAuthEnabled" && isMacMdmEnabledAndConfigured) {
      next.lockEndUserInfo = value as boolean;
    }
    setFormData(next);

    // only set errors that are updates of existing errors
    // new errors are only set onBlur
    const newErrs = validateFormData(next);
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      if (newErrs[k]) {
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onInputBlur = () => {
    setFormErrors(validateFormData(formData));
  };

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    const errs = validateFormData(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }

    setIsUpdating(true);
    const canLockEndUserInfo =
      formData.endUserAuthEnabled && formData.lockEndUserInfo;

    try {
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        enable_end_user_authentication: formData.endUserAuthEnabled,
        // Apple-only fields are omitted when Apple MDM isn't configured.
        ...(isMacMdmEnabledAndConfigured && {
          lock_end_user_info: canLockEndUserInfo,
          enable_managed_local_account: effectiveEnableManagedLocalAccount(
            formData
          ),
          end_user_local_account_type: formData.localAccountType,
        }),
      });
      renderFlash("success", "Successfully updated.");
    } catch {
      renderFlash("error", "Couldn't update settings. Please try again.");
    }

    setIsUpdating(false);
    if (isMacMdmEnabledAndConfigured) {
      setFormData((prev) => ({ ...prev, lockEndUserInfo: canLockEndUserInfo }));
    }
  };

  const sectionProps: IUsersFormSectionProps = {
    formData,
    formErrors,
    onInputChange,
    onInputBlur,
    isIdPConfigured,
    isMacMdmEnabledAndConfigured: !!isMacMdmEnabledAndConfigured,
    gitOpsModeEnabled,
  };

  return (
    <div className={baseClass}>
      <form onSubmit={onSubmit}>
        <LocalAccountSection {...sectionProps} />
        <EndUserAuthSection {...sectionProps} />
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              isLoading={isUpdating}
              type="submit"
            >
              Save
            </Button>
          )}
        />
      </form>
    </div>
  );
};

export default UsersForm;
