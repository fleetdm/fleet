import React, { FormEvent, useState, useEffect, useContext } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { ITeam } from "interfaces/team";
import { IUserFormErrors, UserRole } from "interfaces/user";

import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";
// @ts-ignore
import validPassword from "components/forms/validators/valid_password";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Radio from "components/forms/fields/Radio";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import SelectedTeamsForm from "../SelectedTeamsForm/SelectedTeamsForm";
import SelectRoleForm from "../SelectRoleForm/SelectRoleForm";
import { roleOptions } from "../../helpers/userManagementHelpers";

const baseClass = "create-user-form";

export enum NewUserType {
  AdminInvited = "ADMIN_INVITED",
  AdminCreated = "ADMIN_CREATED",
}

enum UserTeamType {
  GlobalUser = "GLOBAL_USER",
  AssignTeams = "ASSIGN_TEAMS",
}

export interface IFormData {
  email: string;
  name: string;
  newUserType?: NewUserType | null;
  password?: string | null;
  sso_enabled?: boolean;
  global_role: UserRole | null;
  teams: ITeam[];
  currentUserId?: number;
  invited_by?: number;
  role?: UserRole;
}

interface ICreateUserFormProps {
  availableTeams: ITeam[];
  onCancel: () => void;
  onSubmit: (formData: IFormData) => void;
  submitText: string;
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
  isApiOnly?: boolean;
  isNewUser?: boolean;
  isInvitePending?: boolean;
  serverErrors?: { base: string; email: string }; // "server" because this form does its own client validation
  createOrEditUserErrors?: IUserFormErrors;
  isUpdatingUsers?: boolean;
}

const UserForm = ({
  availableTeams,
  onCancel,
  onSubmit,
  submitText,
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
  isApiOnly,
  isNewUser,
  isInvitePending,
  serverErrors,
  createOrEditUserErrors,
  isUpdatingUsers,
}: ICreateUserFormProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [errors, setErrors] = useState<any>(createOrEditUserErrors);
  const [formData, setFormData] = useState<any>({
    email: defaultEmail || "",
    name: defaultName || "",
    newUserType: isNewUser ? NewUserType.AdminCreated : null,
    password: null,
    sso_enabled: isSsoEnabled,
    global_role: defaultGlobalRole || null,
    teams: defaultTeams || [],
    currentUserId,
  });

  const [isGlobalUser, setIsGlobalUser] = useState(!!defaultGlobalRole);

  useEffect(() => {
    setErrors(createOrEditUserErrors);
  }, [createOrEditUserErrors]);

  const onInputChange = (formField: string): ((value: string) => void) => {
    return (value: string) => {
      setErrors({
        ...errors,
        [formField]: null,
      });
      setFormData({
        ...formData,
        [formField]: value,
      });
    };
  };

  const onCheckboxChange = (formField: string): ((evt: string) => void) => {
    return (evt: string) => {
      return onInputChange(formField)(evt);
    };
  };

  const onRadioChange = (formField: string): ((evt: string) => void) => {
    return (evt: string) => {
      return onInputChange(formField)(evt);
    };
  };

  const onIsGlobalUserChange = (value: string): void => {
    const isGlobalUserChange = value === UserTeamType.GlobalUser;
    setIsGlobalUser(isGlobalUserChange);
    setFormData({
      ...formData,
      global_role: isGlobalUserChange ? "observer" : null,
    });
  };

  const onGlobalUserRoleChange = (value: string): void => {
    setFormData({
      ...formData,
      global_role: value,
    });
  };

  const onSelectedTeamChange = (teams: ITeam[]): void => {
    setFormData({
      ...formData,
      teams,
    });
  };

  const onTeamRoleChange = (teams: ITeam[]): void => {
    setFormData({
      ...formData,
      teams,
    });
  };

  const onSsoDisable = (): void => {
    setFormData({
      ...formData,
      sso_enabled: false,
    });
  };

  // UserForm component can be used to create a new user or edit an existing user so submitData will be assembled accordingly
  const createSubmitData = (): IFormData => {
    const submitData = formData;

    if (!isNewUser && !isInvitePending) {
      // if a new password is being set for an existing user, the API expects `new_password` rather than `password`
      submitData.new_password = formData.password;
      delete submitData.password;
      delete submitData.newUserType; // this field will not be submitted when form is used to edit an existing user
      // if an existing user is converted to sso, the API expects `new_password` to be null
      if (formData.sso_enabled) {
        submitData.new_password = null;
      }
    }

    if (
      submitData.sso_enabled ||
      formData.newUserType === NewUserType.AdminInvited
    ) {
      delete submitData.password; // this field will not be submitted with the form
    }

    return isGlobalUser
      ? { ...submitData, global_role: formData.global_role, teams: [] }
      : { ...submitData, global_role: null, teams: formData.teams };
  };

  const validate = (): boolean => {
    if (!validatePresence(formData.name)) {
      setErrors({
        ...errors,
        name: "Name field must be completed",
      });

      return false;
    }

    if (!validatePresence(formData.email)) {
      setErrors({
        ...errors,
        email: "Email field must be completed",
      });

      return false;
    }

    if (!validEmail(formData.email)) {
      setErrors({
        ...errors,
        email: `${formData.email} is not a valid email`,
      });

      return false;
    }

    if (
      !isNewUser &&
      !isInvitePending &&
      formData.password &&
      !validPassword(formData.password) &&
      !formData.sso_enabled
    ) {
      setErrors({
        ...errors,
        password: "Password must meet the criteria below",
      });

      return false;
    }

    if (
      isNewUser &&
      formData.newUserType === NewUserType.AdminCreated &&
      !formData.sso_enabled
    ) {
      if (!validatePresence(formData.password)) {
        setErrors({
          ...errors,
          password: "Password field must be completed",
        });

        return false;
      }
      if (!validPassword(formData.password)) {
        setErrors({
          ...errors,
          password: "Password must meet the criteria below",
        });

        return false;
      }
    }

    if (!formData.global_role && !formData.teams.length) {
      renderFlash("error", `Please select at least one team for this user.`);
      return false;
    }

    return true;
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();
    const valid = validate();
    if (valid) {
      return onSubmit(createSubmitData());
    }
  };

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
            />
          </InfoBanner>
        )}
        <Dropdown
          label="Role"
          value={formData.global_role || "Observer"}
          className={`${baseClass}__global-role-dropdown`}
          options={roleOptions({
            isPremiumTier,
            isApiOnly,
          })}
          searchable={false}
          onChange={onGlobalUserRoleChange}
          wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--global-role`}
        />
      </>
    );
  };

  const renderNoTeamsMessage = (): JSX.Element => {
    return (
      <div>
        <p>
          <strong>You have no teams.</strong>
        </p>
        <p>
          Expecting to see teams? Try again in a few seconds as the system
          catches up or&nbsp;
          <Link
            className={`${baseClass}__create-team-link`}
            to={PATHS.ADMIN_TEAMS}
          >
            create a team
          </Link>
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
              <InfoBanner className={`${baseClass}__user-permissions-info`}>
                <p>
                  Users can manage or observe team-specific users, entities, and
                  settings in Fleet.
                </p>
                <CustomLink
                  url="https://fleetdm.com/docs/using-fleet/permissions#team-member-permissions"
                  text="Learn more about user permissions"
                  newTab
                />
              </InfoBanner>
              <SelectedTeamsForm
                availableTeams={availableTeams}
                usersCurrentTeams={formData.teams}
                onFormChange={onSelectedTeamChange}
                isApiOnly={isApiOnly}
              />
            </>
          ) : (
            <SelectRoleForm
              label="Role"
              currentTeam={currentTeam || formData.teams[0]}
              teams={formData.teams}
              defaultTeamRole={defaultTeamRole || "observer"}
              onFormChange={onTeamRoleChange}
              isApiOnly={isApiOnly}
            />
          ))}
        {!availableTeams.length && renderNoTeamsMessage()}
      </>
    );
  };

  if (!isPremiumTier && !isGlobalUser) {
    console.log(
      `Note: Fleet Free UI does not have teams options.\n
        User ${formData.name} is already assigned to a team and cannot be reassigned without access to Fleet Premium UI.`
    );
  }

  return (
    <form className={baseClass} autoComplete="off">
      {/* {baseError && <div className="form__base-error">{baseError}</div>} */}
      <InputField
        label="Full name"
        autofocus
        error={errors.name}
        name="name"
        onChange={onInputChange("name")}
        placeholder="Full name"
        value={formData.name || ""}
        inputOptions={{
          maxLength: "80",
        }}
      />
      <InputField
        label="Email"
        error={errors.email || serverErrors?.email}
        name="email"
        onChange={onInputChange("email")}
        placeholder="Email"
        value={formData.email || ""}
        disabled={!isNewUser && !(smtpConfigured || sesConfigured)}
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
      {!isNewUser &&
        !isInvitePending &&
        isModifiedByGlobalAdmin &&
        !formData.sso_enabled && (
          <div className={`${baseClass}__edit-password`}>
            <div className={`${baseClass}__password`}>
              <InputField
                label="Password"
                error={errors.password}
                name="password"
                onChange={onInputChange("password")}
                placeholder="••••••••"
                value={formData.password || ""}
                type="password"
                helpText="12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#)."
                blockAutoComplete
              />
            </div>
          </div>
        )}
      {((!isNewUser && formData.sso_enabled) || canUseSso) && (
        <div className={`${baseClass}__sso-input`}>
          <Checkbox
            name="sso_enabled"
            onChange={onCheckboxChange("sso_enabled")}
            value={formData.sso_enabled}
            disabled={!canUseSso}
            wrapperClassName={`${baseClass}__invite-admin`}
            helpText={
              canUseSso ? (
                <p className={`${baseClass}__sso-input sublabel`}>
                  Password authentication will be disabled for this user.
                </p>
              ) : (
                <span className={`${baseClass}__sso-input sublabel-nosso`}>
                  This user previously signed in via SSO, which has been
                  globally disabled.{" "}
                  <button className="button--text-link" onClick={onSsoDisable}>
                    Add password instead
                    <Icon
                      name="chevron-right"
                      color="core-fleet-blue"
                      size="small"
                    />
                  </button>
                </span>
              )
            }
          >
            <span
              className={!canUseSso ? `${baseClass}__sso-input--disabled` : ""}
            >
              Enable single sign-on
            </span>
          </Checkbox>
        </div>
      )}
      {isNewUser && (
        <>
          <div className="form-field">
            {isModifiedByGlobalAdmin ? (
              <>
                <div className="form-field__label">Account</div>
                <Radio
                  className={`${baseClass}__radio-input`}
                  label="Create user"
                  id="create-user"
                  checked={formData.newUserType !== NewUserType.AdminInvited}
                  value={NewUserType.AdminCreated}
                  name="newUserType"
                  onChange={onRadioChange("newUserType")}
                />
                <Radio
                  className={`${baseClass}__radio-input`}
                  label="Invite user"
                  id="invite-user"
                  disabled={!(smtpConfigured || sesConfigured)}
                  checked={formData.newUserType === NewUserType.AdminInvited}
                  value={NewUserType.AdminInvited}
                  name="newUserType"
                  onChange={onRadioChange("newUserType")}
                  tooltip={
                    smtpConfigured || sesConfigured ? (
                      ""
                    ) : (
                      <>
                        The &quot;Invite user&quot; feature requires that SMTP
                        or SES is configured in order to send invitation emails.
                        <br />
                        <br />
                        SMTP can be configured in Settings &gt; Organization
                        settings.
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
                name="newUserType"
              />
            )}
          </div>
          {formData.newUserType !== NewUserType.AdminInvited &&
            !formData.sso_enabled && (
              <>
                <div className={`${baseClass}__password`}>
                  <InputField
                    label="Password"
                    error={errors.password}
                    name="password"
                    onChange={onInputChange("password")}
                    placeholder="Password"
                    value={formData.password || ""}
                    type="password"
                    helpText="12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#)."
                    blockAutoComplete
                    tooltip={
                      <>
                        This password is temporary. This user will be asked to
                        set a new password after logging in to the Fleet UI.
                        <br />
                        <br />
                        This user will not be asked to set a new password after
                        logging in to fleetctl or the Fleet API.
                      </>
                    }
                  />
                </div>
              </>
            )}
        </>
      )}
      {isPremiumTier && (
        <>
          <div className="form-field">
            <div className="form-field__label">Team</div>
            {isModifiedByGlobalAdmin ? (
              <>
                <Radio
                  className={`${baseClass}__radio-input`}
                  label="Global user"
                  id="global-user"
                  checked={isGlobalUser}
                  value={UserTeamType.GlobalUser}
                  name="userTeamType"
                  onChange={onIsGlobalUserChange}
                />
                <Radio
                  className={`${baseClass}__radio-input`}
                  label="Assign teams"
                  id="assign-teams"
                  checked={!isGlobalUser}
                  value={UserTeamType.AssignTeams}
                  name="userTeamType"
                  onChange={onIsGlobalUserChange}
                  disabled={!availableTeams.length}
                />
              </>
            ) : (
              <>{currentTeam ? currentTeam.name : ""}</>
            )}
          </div>
          {isGlobalUser ? renderGlobalRoleForm() : renderTeamsForm()}
        </>
      )}
      {!isPremiumTier && renderGlobalRoleForm()}

      <div className="modal-cta-wrap">
        <Button
          type="submit"
          variant="brand"
          onClick={onFormSubmit}
          className={`${submitText === "Create" ? "create" : "save"}-loading
          `}
          isLoading={isUpdatingUsers}
        >
          {submitText}
        </Button>
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default UserForm;
