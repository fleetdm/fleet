import React, { FormEvent, useState, useEffect, useContext } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { ITeam } from "interfaces/team";
import { IUserFormErrors } from "interfaces/user";

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
import SelectedTeamsForm from "../SelectedTeamsForm/SelectedTeamsForm";
import SelectRoleForm from "../SelectRoleForm/SelectRoleForm";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "create-user-form";

export enum NewUserType {
  AdminInvited = "ADMIN_INVITED",
  AdminCreated = "ADMIN_CREATED",
}

enum UserTeamType {
  GlobalUser = "GLOBAL_USER",
  AssignTeams = "ASSIGN_TEAMS",
}

const globalUserRoles = [
  {
    disabled: false,
    label: "Observer",
    value: "observer",
  },
  {
    disabled: false,
    label: "Maintainer",
    value: "maintainer",
  },
  {
    disabled: false,
    label: "Admin",
    value: "admin",
  },
];

export interface IFormData {
  email: string;
  name: string;
  newUserType?: NewUserType | null;
  password?: string | null;
  sso_enabled?: boolean;
  global_role: string | null;
  teams: ITeam[];
  currentUserId?: number;
  invited_by?: number;
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
  defaultGlobalRole?: string | null;
  defaultTeamRole?: string;
  defaultTeams?: ITeam[];
  isPremiumTier: boolean;
  smtpConfigured?: boolean;
  canUseSso: boolean; // corresponds to whether SSO is enabled for the organization
  isSsoEnabled?: boolean; // corresponds to whether SSO is enabled for the individual user
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
  canUseSso,
  isSsoEnabled,
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

  const [isGlobalUser, setIsGlobalUser] = useState<boolean>(
    !!defaultGlobalRole
  );

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
              Global users can only be members of the top level team and can
              manage or observe all users, entities, and settings in Fleet.
            </p>
            <a
              href="https://fleetdm.com/docs/using-fleet/permissions#user-permissions"
              target="_blank"
              rel="noopener noreferrer"
            >
              Learn more about user permissions
              <img src={OpenNewTabIcon} alt="open new tab" />
            </a>
          </InfoBanner>
        )}
        <Dropdown
          label="Role"
          value={formData.global_role || "Observer"}
          className={`${baseClass}__global-role-dropdown`}
          options={globalUserRoles}
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
                  Users can be members of multiple teams and can only manage or
                  observe team-specific users, entities, and settings in Fleet.
                </p>
                <a
                  href="https://fleetdm.com/docs/using-fleet/permissions#team-member-permissions"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Learn more about user permissions
                  <img src={OpenNewTabIcon} alt="open new tab" />
                </a>
              </InfoBanner>
              <SelectedTeamsForm
                availableTeams={availableTeams}
                usersCurrentTeams={formData.teams}
                onFormChange={onSelectedTeamChange}
              />
            </>
          ) : (
            <SelectRoleForm
              label="Role"
              currentTeam={currentTeam || formData.teams[0]}
              teams={formData.teams}
              defaultTeamRole={defaultTeamRole || "observer"}
              onFormChange={onTeamRoleChange}
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
        disabled={!isNewUser && !smtpConfigured}
        tooltip={
          "\
              Editing an email address requires that SMTP is configured in order to send a validation email. \
              <br /><br /> \
              Users with Admin role can configure SMTP in <strong>Settings &gt; Organization settings</strong>. \
            "
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
                hint={[
                  "Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)",
                ]}
                blockAutoComplete
              />
            </div>
          </div>
        )}
      <div className={`${baseClass}__sso-input`}>
        <Checkbox
          name="sso_enabled"
          onChange={onCheckboxChange("sso_enabled")}
          value={canUseSso && formData.sso_enabled}
          disabled={!canUseSso}
          wrapperClassName={`${baseClass}__invite-admin`}
          tooltip={`
              Enabling single sign-on for a user requires that SSO is first enabled for the organization.
              <br /><br />
              Users with Admin role can configure SSO in <strong>Settings &gt; Organization settings</strong>.
            `}
        >
          Enable single sign-on
        </Checkbox>
        <p className={`${baseClass}__sso-input sublabel`}>
          Password authentication will be disabled for this user.
        </p>
      </div>
      {isNewUser && (
        <div className={`${baseClass}__new-user-container`}>
          <div className={`${baseClass}__new-user-radios`}>
            {isModifiedByGlobalAdmin ? (
              <>
                <Radio
                  className={`${baseClass}__radio-input`}
                  label={"Create user"}
                  id={"create-user"}
                  checked={formData.newUserType !== NewUserType.AdminInvited}
                  value={NewUserType.AdminCreated}
                  name={"newUserType"}
                  onChange={onRadioChange("newUserType")}
                />
                <Radio
                  className={`${baseClass}__radio-input`}
                  label={"Invite user"}
                  id={"invite-user"}
                  disabled={!smtpConfigured}
                  checked={formData.newUserType === NewUserType.AdminInvited}
                  value={NewUserType.AdminInvited}
                  name={"newUserType"}
                  onChange={onRadioChange("newUserType")}
                  tooltip={
                    smtpConfigured
                      ? ""
                      : `
                      The &quot;Invite user&quot; feature requires that SMTP
                      is configured in order to send invitation emails.
                      <br /><br />
                      SMTP can be configured in Settings &gt; Organization settings.
                    `
                  }
                />
              </>
            ) : (
              <input
                type="hidden"
                id={"create-user"}
                value={NewUserType.AdminCreated}
                name={"newUserType"}
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
                    hint={[
                      "Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)",
                    ]}
                    blockAutoComplete
                    tooltip={`\
                      This password is temporary. This user will be asked to set a new password after logging in to the Fleet UI.<br /><br />\
                      This user will not be asked to set a new password after logging in to fleetctl or the Fleet API.\
                    `}
                  />
                </div>
              </>
            )}
        </div>
      )}
      {isPremiumTier && (
        <div className={`${baseClass}__selected-teams-container`}>
          <div className={`${baseClass}__team-radios`}>
            <p className={`${baseClass}__label`}>Team</p>
            {isModifiedByGlobalAdmin ? (
              <>
                <Radio
                  className={`${baseClass}__radio-input`}
                  label={"Global user"}
                  id={"global-user"}
                  checked={isGlobalUser}
                  value={UserTeamType.GlobalUser}
                  name={"userTeamType"}
                  onChange={onIsGlobalUserChange}
                />
                <Radio
                  className={`${baseClass}__radio-input`}
                  label={"Assign teams"}
                  id={"assign-teams"}
                  checked={!isGlobalUser}
                  value={UserTeamType.AssignTeams}
                  name={"userTeamType"}
                  onChange={onIsGlobalUserChange}
                  disabled={!availableTeams.length}
                />
              </>
            ) : (
              <p className="current-team">
                {currentTeam ? currentTeam.name : ""}
              </p>
            )}
          </div>
          <div className={`${baseClass}__teams-form-container`}>
            {isGlobalUser ? renderGlobalRoleForm() : renderTeamsForm()}
          </div>
        </div>
      )}
      {!isPremiumTier && renderGlobalRoleForm()}

      <div className="modal-cta-wrap">
        <Button
          type="submit"
          variant="brand"
          onClick={onFormSubmit}
          spinner={isUpdatingUsers}
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
