import React, { useContext, useEffect, useId } from "react";
import PATHS from "router/paths";

import { PRIMO_TOOLTIP } from "utilities/constants";

import { AppContext } from "context/app";

import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

import { SingleValue } from "react-select-5";
import Button from "components/buttons/Button";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import ModalFooter from "components/ModalFooter";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";
// @ts-ignore
import validPassword from "components/forms/validators/valid_password";
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";
import Radio from "components/forms/fields/Radio";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import SelectedTeamsForm from "../SelectedTeamsForm/SelectedTeamsForm";
import SelectRoleForm from "../SelectRoleForm/SelectRoleForm";
import { roleOptions } from "../../helpers/userManagementHelpers";

const baseClass = "user-form";

const PASSWORD_ERRORS: Record<string, string> = {
  too_short: "Enter a password with at least 12 characters",
  too_long: "Enter a password with 48 characters or fewer",
  invalid_format:
    "Enter a password with at least 1 letter, 1 number and 1 symbol",
};

export enum NewUserType {
  AdminInvited = "ADMIN_INVITED",
  AdminCreated = "ADMIN_CREATED",
}

enum UserTeamType {
  GlobalUser = "GLOBAL_USER",
  AssignTeams = "ASSIGN_TEAMS",
}

/** What this form submits. Shaped for the users/invites APIs. */
export interface IUserFormData {
  email: string;
  name: string;
  newUserType?: NewUserType | null;
  password?: string | null;
  new_password?: string | null; // if a new password is being set for an existing user, the API expects `new_password` rather than `password`
  sso_enabled: boolean;
  mfa_enabled?: boolean;
  global_role: UserRole | null;
  teams: ITeam[];
  currentUserId?: number;
  invited_by?: number;
  role?: UserRole;
}

/** What the fields on screen hold. */
type UserFormState = {
  email: string;
  name: string;
  newUserType: NewUserType | null;
  password: string;
  sso_enabled: boolean;
  mfa_enabled: boolean;
  global_role: UserRole | null;
  teams: ITeam[];
};

interface IUserFormProps {
  availableTeams: ITeam[];
  onCancel: () => void;
  /** Return the request promise to have the hook track in-flight state. */
  onSubmit: (formData: IUserFormData) => void | Promise<unknown>;
  defaultName?: string;
  defaultEmail?: string;
  currentUserId?: number;
  currentTeam?: ITeam;
  isModifiedByGlobalAdmin?: boolean | false;
  defaultGlobalRole?: UserRole | null;
  defaultTeamRole?: UserRole;
  defaultTeams?: ITeam[];
  isPremiumTier: boolean;
  smtpConfigured?: boolean;
  sesConfigured?: boolean;
  canUseSso: boolean; // corresponds to whether SSO is enabled for the organization
  isSsoEnabled?: boolean; // corresponds to whether SSO is enabled for the individual user
  isMfaEnabled?: boolean; // corresponds to whether MFA is enabled for the individual user
  isApiOnly?: boolean;
  isNewUser?: boolean;
  isInvitePending?: boolean;
  /** Field-specific errors from the API. Pass a new object per failed request. */
  serverErrors: IFormErrors;
  isUpdatingUsers?: boolean;
}

const UserForm = ({
  availableTeams,
  onCancel,
  onSubmit,
  defaultName,
  defaultEmail,
  currentUserId,
  currentTeam,
  isModifiedByGlobalAdmin,
  defaultGlobalRole,
  defaultTeamRole,
  defaultTeams,
  isPremiumTier,
  smtpConfigured,
  sesConfigured,
  canUseSso,
  isSsoEnabled,
  isMfaEnabled,
  isApiOnly,
  isNewUser = false,
  isInvitePending,
  serverErrors,
  isUpdatingUsers,
}: IUserFormProps): JSX.Element => {
  const { config } = useContext(AppContext);
  const priMode = config?.partnerships?.enable_primo;

  // The submit button lives in a ModalFooter outside the <form>, so it reaches
  // the form's onSubmit through this id rather than through its own onClick.
  // Per instance, so two forms on one page can't collide on the id.
  const formId = useId();

  // Mirrors the render condition for renderPasswordSection — a hidden field is
  // never validated.
  const isPasswordShown = (data: UserFormState) =>
    ((isNewUser && data.newUserType !== NewUserType.AdminInvited) ||
      (!isNewUser && !isInvitePending)) &&
    !data.sso_enabled;

  // A password is required when creating a user with SSO off, and when moving a
  // user off SSO onto password auth (which includes the case where SSO was
  // turned off org-wide). On an edit form for a password user it stays optional:
  // leaving it blank keeps the current password.
  const isPasswordRequired = (data: UserFormState) =>
    (isNewUser && data.newUserType === NewUserType.AdminCreated) ||
    !!isSsoEnabled;

  const validate = (data: UserFormState): IFormErrors => {
    const errors: IFormErrors = {};

    if (!validatePresence(data.name)) {
      errors.name = "Enter a name";
    }

    if (!validatePresence(data.email)) {
      errors.email = "Enter an email";
    } else if (!validEmail(data.email)) {
      errors.email = "Enter a valid email";
    }

    if (isPasswordShown(data)) {
      const hasPassword = validatePresence(data.password);
      if (!hasPassword && isPasswordRequired(data)) {
        errors.password = "Enter a password";
      } else if (hasPassword) {
        // An optional password still has to satisfy the format rule once it has
        // a value. Fall back to the validator's own copy if it grows a new code.
        const { error_code: errorCode, error } = validPassword(data.password);
        if (errorCode) {
          errors.password = PASSWORD_ERRORS[errorCode] || error;
        }
      }
    }

    // The fleet selector is on screen whenever the user isn't global. Its
    // "at least one" error belongs on the selector, not in a toast.
    if (isPremiumTier && !data.global_role && !data.teams.length) {
      errors.teams = "Select at least one fleet";
    }

    return errors;
  };

  const {
    formData,
    setField,
    commitFields,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<UserFormState>({
    initialFormData: {
      email: defaultEmail || "",
      name: defaultName || "",
      newUserType: isNewUser ? NewUserType.AdminCreated : null,
      password: "",
      sso_enabled: isSsoEnabled || false,
      mfa_enabled: isMfaEnabled || false,
      global_role: defaultGlobalRole || null,
      teams: defaultTeams || [],
    },
    validate,
    serverErrors,
    isSubmitting: isUpdatingUsers,
    skipTrim: ["password"],
  });

  const isGlobalUser = formData.global_role !== null;

  // watch for population from context, set state
  useEffect(() => {
    if (priMode) {
      commitFields({ global_role: "observer", teams: [] });
    }
  }, [priMode, commitFields]);

  useEffect(() => {
    // If SSO is globally disabled but user previously signed in via SSO,
    // require password is automatically selected on first render
    if (!canUseSso && !isNewUser && isSsoEnabled) {
      commitFields({ sso_enabled: false });
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const onGlobalUserRoleChange = (selected: SingleValue<CustomOptionType>) => {
    if (selected) {
      commitFields({ global_role: selected.value as UserRole });
    }
  };

  const onIsGlobalUserChange = (value: string): void => {
    commitFields({
      global_role: value === UserTeamType.GlobalUser ? "observer" : null,
    });
  };

  // UserForm can add a new user or edit an existing one, so the payload is
  // assembled accordingly.
  const buildSubmitData = (data: UserFormState): IUserFormData => {
    const submitData: IUserFormData = {
      email: data.email,
      name: data.name,
      newUserType: data.newUserType,
      password: data.password,
      sso_enabled: data.sso_enabled,
      mfa_enabled: data.mfa_enabled,
      global_role: data.global_role,
      teams: data.teams,
      currentUserId,
    };

    if (!isNewUser && !isInvitePending) {
      // if a new password is being set for an existing user, the API expects `new_password` rather than `password`
      submitData.new_password = data.password;
      submitData.password = null;
      delete submitData.newUserType; // this field will not be submitted when form is used to edit an existing user
      // if an existing user is converted to sso, the API expects `new_password` to be null
      if (data.sso_enabled) {
        submitData.new_password = null;
        submitData.mfa_enabled = false; // Edge case a user sets mfa, and then sets sso, we need to remove mfa
      }
    }

    if (
      submitData.sso_enabled ||
      data.newUserType === NewUserType.AdminInvited
    ) {
      submitData.password = null; // this field will not be submitted with the form
    }

    return data.global_role !== null
      ? { ...submitData, global_role: data.global_role, teams: [] }
      : { ...submitData, global_role: null, teams: data.teams };
  };

  const onValidSubmit = (data: UserFormState) =>
    onSubmit(buildSubmitData(data));

  const renderGlobalRoleForm = (): JSX.Element => {
    return (
      <>
        {isPremiumTier && (
          <InfoBanner className={`${baseClass}__user-permissions-info`}>
            <p>
              Global users can manage or observe all users, entities, and
              settings in Fleet.
            </p>
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/permissions#user-permissions"
              text="Learn more about user permissions"
              newTab
              variant="banner-link"
            />
          </InfoBanner>
        )}
        <DropdownWrapper
          label="Role"
          name="Role"
          className={`${baseClass}__global-role-dropdown`}
          options={roleOptions({ isPremiumTier, isApiOnly })}
          value={formData.global_role || "Observer"}
          onChange={onGlobalUserRoleChange}
          isSearchable={false}
          isDisabled={isSubmitting}
        />
      </>
    );
  };

  const renderNoTeamsMessage = (): JSX.Element => {
    return (
      <div>
        <p>
          <strong>You have no fleets.</strong>
        </p>
        <p>
          Expecting to see fleets? Try again in a few seconds as the system
          catches up or&nbsp;
          <CustomLink
            className={`${baseClass}__create-team-link`}
            url={PATHS.ADMIN_FLEETS}
            text="create a fleet"
          />
          .
        </p>
      </div>
    );
  };

  const renderTeamsForm = (): JSX.Element => {
    return (
      <>
        {!!availableTeams.length &&
          (isModifiedByGlobalAdmin ? (
            <>
              <InfoBanner
                color="grey"
                className={`${baseClass}__user-permissions-info`}
              >
                <p>
                  Users can manage or observe fleet-specific users, entities,
                  and settings in Fleet.
                </p>
                <CustomLink
                  url="https://fleetdm.com/docs/using-fleet/permissions#team-member-permissions"
                  text="Learn more about user permissions"
                  newTab
                  variant="banner-link"
                />
              </InfoBanner>
              <SelectedTeamsForm
                availableTeams={availableTeams}
                usersCurrentTeams={formData.teams}
                onFormChange={(teams: ITeam[]) => commitFields({ teams })}
                isApiOnly={isApiOnly}
              />
            </>
          ) : (
            <SelectRoleForm
              currentTeam={currentTeam || formData.teams[0]}
              teams={formData.teams}
              defaultTeamRole={defaultTeamRole || "Observer"}
              onFormChange={(teams: ITeam[]) => commitFields({ teams })}
              isApiOnly={isApiOnly}
            />
          ))}
        {getError("teams") && (
          <div className="form-field__label form-field__label--error">
            {getError("teams")}
          </div>
        )}
        {!availableTeams.length && renderNoTeamsMessage()}
      </>
    );
  };

  if (!isPremiumTier && !isGlobalUser) {
    console.log(
      `Note: Fleet Free UI does not have fleets options.\n
        User ${formData.name} is already assigned to a fleet and cannot be reassigned without access to Fleet Premium UI.`
    );
  }

  const renderAccountSection = () => (
    <div className="form-field">
      {isModifiedByGlobalAdmin ? (
        <>
          <div className="form-field__label">Account</div>
          <Radio
            className={`${baseClass}__radio-input`}
            label="Add user"
            id="create-user"
            checked={formData.newUserType !== NewUserType.AdminInvited}
            value={NewUserType.AdminCreated}
            name="new-user-type"
            onChange={(value: string) =>
              commitFields({ newUserType: value as NewUserType })
            }
            disabled={isSubmitting}
          />
          <Radio
            className={`${baseClass}__radio-input`}
            label="Invite user"
            id="invite-user"
            disabled={isSubmitting || !(smtpConfigured || sesConfigured)}
            checked={formData.newUserType === NewUserType.AdminInvited}
            value={NewUserType.AdminInvited}
            name="new-user-type"
            onChange={(value: string) =>
              commitFields({ newUserType: value as NewUserType })
            }
            tooltip={
              smtpConfigured || sesConfigured ? (
                ""
              ) : (
                <>
                  The &quot;Invite user&quot; feature requires that SMTP or SES
                  is configured in order to send invitation emails.
                  <br />
                  <br />
                  SMTP can be configured in Settings &gt; Organization settings.
                </>
              )
            }
          />
        </>
      ) : (
        <input
          type="hidden"
          id="create-user"
          value={NewUserType.AdminCreated}
          name="new-user-type"
        />
      )}
    </div>
  );

  const renderNameAndEmailSection = () => (
    <>
      <InputField
        label="Full name"
        autofocus
        error={getError("name")}
        name="name"
        onChange={(value: string) => setField("name", value)}
        onFocus={() => clearFieldError("name")}
        onBlur={() => validateField("name")}
        placeholder="Full name"
        value={formData.name}
        disabled={isSubmitting}
        inputOptions={{
          maxLength: 80,
        }}
      />
      <InputField
        label="Email"
        error={getError("email")}
        name="email"
        type="email"
        onChange={(value: string) => setField("email", value)}
        onFocus={() => clearFieldError("email")}
        onBlur={() => validateField("email")}
        placeholder="Email"
        value={formData.email}
        disabled={isSubmitting}
        readOnly={!isNewUser && !(smtpConfigured || sesConfigured)}
        tooltip={
          <>
            Editing an email address requires that SMTP or SES is configured in
            order to send a validation email.
            <br />
            <br />
            Users with Admin role can configure SMTP in{" "}
            <strong>Settings &gt; Organization settings</strong>.
          </>
        }
      />
    </>
  );

  const renderAuthenticationSection = () => (
    <div className="form-field">
      <div className="form-field__label">Authentication</div>
      <Radio
        className={`${baseClass}__radio-input`}
        label={
          canUseSso ? (
            "Single sign-on"
          ) : (
            <TooltipWrapper
              tipContent={
                <>
                  SSO is not enabled in organization settings.
                  <br />
                  User must sign in with a password.
                </>
              }
            >
              Single sign-on
            </TooltipWrapper>
          )
        }
        id="single-sign-on-authentication"
        checked={!!formData.sso_enabled}
        value="true"
        name="authentication-type"
        onChange={() => commitFields({ sso_enabled: true })}
        disabled={isSubmitting || !canUseSso}
      />
      <Radio
        className={`${baseClass}__radio-input`}
        label="Password"
        id="password-authentication"
        // allow the user to change auth back to password if they only changed the form to SSO in
        // the current session, that is, in the db, the user is still password authenticated
        checked={!formData.sso_enabled}
        value="false"
        name="authentication-type"
        onChange={() => commitFields({ sso_enabled: false })}
        disabled={isSubmitting}
      />
    </div>
  );

  const renderPasswordSection = () => (
    <div className={`${baseClass}__${isNewUser ? "" : "edit-"}password`}>
      <InputField
        label="Password"
        error={getError("password")}
        name="password"
        onChange={(value: string) => setField("password", value)}
        onFocus={() => clearFieldError("password")}
        onBlur={() => validateField("password")}
        placeholder={isNewUser ? "Password" : "••••••••"}
        value={formData.password}
        type="password"
        disabled={isSubmitting}
        helpText="12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#)."
        blockAutoComplete
        tooltip={
          isNewUser ? (
            <>
              This password is temporary. This user will be asked to set a new
              password after logging in to the Fleet UI.
              <br />
              <br />
              This user will not be asked to set a new password after logging in
              to fleetctl or the Fleet API.
            </>
          ) : undefined
        }
      />
    </div>
  );

  // 2fa option shows on premium tier or if previously set to true before downgrading to free
  const renderTwoFactorAuthenticationOption = () => (
    <div className="form-field">
      {/* Renders missing password heading when inviting a user */}
      {formData.newUserType === NewUserType.AdminInvited && (
        <div className="form-field__label">Password</div>
      )}
      <Checkbox
        name="mfa_enabled"
        onChange={(value: boolean) => commitFields({ mfa_enabled: value })}
        value={formData.mfa_enabled}
        wrapperClassName={`${baseClass}__2fa`}
        helpText="User will be asked to authenticate with a magic link that will be sent to their email."
        disabled={isSubmitting || (!smtpConfigured && !sesConfigured)}
      >
        {smtpConfigured || sesConfigured ? (
          "Enable two-factor authentication (email)"
        ) : (
          <TooltipWrapper
            tipContent={
              <>
                This feature requires that SMTP or SES is configured in order to
                send authentication emails.
                <br />
                <br />
                SMTP can be configured in Settings &gt; Organization settings.
              </>
            }
          >
            Enable two-factor authentication (email)
          </TooltipWrapper>
        )}
      </Checkbox>
    </div>
  );

  const renderGlobalAdminOptions = () => {
    if (priMode) {
      return (
        <TooltipWrapper
          tipContent={PRIMO_TOOLTIP}
          tipOffset={20}
          position="right"
          showArrow
          underline={false}
        >
          <Radio
            className={`${baseClass}__radio-input`}
            label="Global user"
            id="global-user"
            checked={isGlobalUser}
            value={UserTeamType.GlobalUser}
            name="user-team-type"
            onChange={onIsGlobalUserChange}
            disabled
          />
          <Radio
            className={`${baseClass}__radio-input`}
            label="Assign to fleet(s)"
            id="assign-teams"
            checked={!isGlobalUser}
            value={UserTeamType.AssignTeams}
            name="user-team-type"
            onChange={onIsGlobalUserChange}
            disabled
          />
        </TooltipWrapper>
      );
    }
    return (
      <>
        <Radio
          className={`${baseClass}__radio-input`}
          label="Global user"
          id="global-user"
          checked={isGlobalUser}
          value={UserTeamType.GlobalUser}
          name="user-team-type"
          onChange={onIsGlobalUserChange}
          disabled={isSubmitting}
        />
        <Radio
          className={`${baseClass}__radio-input`}
          label={`Assign to fleet(s)`}
          id="assign-teams"
          checked={!isGlobalUser}
          value={UserTeamType.AssignTeams}
          name="user-team-type"
          onChange={onIsGlobalUserChange}
          disabled={isSubmitting || !availableTeams.length}
        />
      </>
    );
  };

  const renderPremiumRoleOptions = () => (
    <>
      <div className="form-field team-field">
        <div className="form-field__label">Permissions</div>
        {isModifiedByGlobalAdmin ? (
          renderGlobalAdminOptions()
        ) : (
          <>{currentTeam ? currentTeam.name : ""}</>
        )}
      </div>
      {isGlobalUser ? renderGlobalRoleForm() : renderTeamsForm()}
    </>
  );

  const renderFormContent = () => {
    return (
      <div className={baseClass}>
        <form
          autoComplete="off"
          id={formId}
          onSubmit={handleSubmit(onValidSubmit)}
        >
          {isNewUser && renderAccountSection()}
          {renderNameAndEmailSection()}
          {renderAuthenticationSection()}
          {isPasswordShown(formData) && renderPasswordSection()}
          {(isPremiumTier || isMfaEnabled) &&
            !formData.sso_enabled &&
            renderTwoFactorAuthenticationOption()}
          {isPremiumTier ? renderPremiumRoleOptions() : renderGlobalRoleForm()}
        </form>
      </div>
    );
  };

  const renderFooter = () => (
    <ModalFooter
      primaryButtons={
        <>
          <Button onClick={onCancel} variant="secondary">
            Cancel
          </Button>
          <Button
            type="submit"
            formId={formId}
            className={`${isNewUser ? "add" : "save"}-loading
          `}
            isLoading={isSubmitting}
            disabled={isSubmitting}
          >
            {isNewUser ? "Add" : "Save"}
          </Button>
        </>
      }
    />
  );

  return (
    <>
      {renderFormContent()}
      {renderFooter()}
    </>
  );
};

export default UserForm;
