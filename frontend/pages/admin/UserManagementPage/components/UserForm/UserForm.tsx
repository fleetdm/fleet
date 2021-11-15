import React, { Component, FormEvent } from "react";
import ReactTooltip from "react-tooltip";
import { Link } from "react-router";
import PATHS from "router/paths";

import { ITeam } from "interfaces/team";
import { IUserFormErrors } from "interfaces/user";
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import validPassword from "components/forms/validators/valid_password";
// @ts-ignore
import IconToolTip from "components/IconToolTip";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Radio from "components/forms/fields/Radio";
import InfoBanner from "components/InfoBanner/InfoBanner";
import SelectedTeamsForm from "../SelectedTeamsForm/SelectedTeamsForm";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";
import SelectRoleForm from "../SelectRoleForm/SelectRoleForm";

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
  serverErrors?: IUserFormErrors; // "server" because this form does its own client validation
}

interface ICreateUserFormState {
  errors: IUserFormErrors;
  formData: IFormData;
  isGlobalUser: boolean;
}

class UserForm extends Component<ICreateUserFormProps, ICreateUserFormState> {
  constructor(props: ICreateUserFormProps) {
    super(props);

    this.state = {
      errors: {
        email: null,
        name: null,
        password: null,
        sso_enabled: null,
      },
      formData: {
        email: props.defaultEmail || "",
        name: props.defaultName || "",
        newUserType: props.isNewUser ? NewUserType.AdminCreated : null,
        password: null,
        sso_enabled: props.isSsoEnabled,
        global_role: props.defaultGlobalRole || null,
        teams: props.defaultTeams || [],
        currentUserId: props.currentUserId,
      },
      isGlobalUser: props.defaultGlobalRole !== null,
    };
  }

  onInputChange = (formField: string): ((value: string) => void) => {
    return (value: string) => {
      const { errors, formData } = this.state;

      this.setState({
        errors: {
          ...errors,
          [formField]: null,
        },
        formData: {
          ...formData,
          [formField]: value,
        },
      });
    };
  };

  onCheckboxChange = (formField: string): ((evt: string) => void) => {
    return (evt: string) => {
      return this.onInputChange(formField)(evt);
    };
  };

  onRadioChange = (formField: string): ((evt: string) => void) => {
    return (evt: string) => {
      return this.onInputChange(formField)(evt);
    };
  };

  onIsGlobalUserChange = (value: string): void => {
    const { formData } = this.state;
    const isGlobalUser = value === UserTeamType.GlobalUser;
    this.setState({
      isGlobalUser,
      formData: {
        ...formData,
        global_role: isGlobalUser ? "observer" : null,
      },
    });
  };

  onGlobalUserRoleChange = (value: string): void => {
    const { formData } = this.state;
    this.setState({
      formData: {
        ...formData,
        global_role: value,
      },
    });
  };

  onSelectedTeamChange = (teams: ITeam[]): void => {
    const { formData } = this.state;
    this.setState({
      formData: {
        ...formData,
        teams,
      },
    });
  };

  onTeamRoleChange = (teams: ITeam[]): void => {
    const { formData } = this.state;
    this.setState({
      formData: {
        ...formData,
        teams,
      },
    });
  };

  onFormSubmit = (evt: FormEvent): void => {
    const { createSubmitData, validate } = this;
    evt.preventDefault();
    const valid = validate();
    if (valid) {
      const { onSubmit } = this.props;
      return onSubmit(createSubmitData());
    }
  };

  // UserForm component can be used to create a new user or edit an existing user so submitData will be assembled accordingly
  createSubmitData = (): IFormData => {
    const { currentUserId, isNewUser } = this.props;
    const {
      isGlobalUser,
      formData: {
        email,
        name,
        newUserType,
        password,
        sso_enabled,
        global_role,
        teams,
      },
    } = this.state;

    const submitData = {
      email,
      name,
      newUserType,
      password,
      sso_enabled,
      currentUserId,
    };

    if (!isNewUser) {
      delete submitData.newUserType; // this field will not be submitted when form is used to edit an existing user
    }

    if (submitData.sso_enabled || newUserType === NewUserType.AdminInvited) {
      delete submitData.password; // this field will not be submitted with the form
    }

    return isGlobalUser
      ? { ...submitData, global_role, teams: [] }
      : { ...submitData, global_role: null, teams };
  };

  validate = (): boolean => {
    const {
      errors,
      formData: { email, password, newUserType, sso_enabled },
    } = this.state;
    const { isNewUser } = this.props;

    if (!validatePresence(email)) {
      this.setState({
        errors: {
          ...errors,
          email: "Email field must be completed",
        },
      });

      return false;
    }

    if (!validEmail(email)) {
      this.setState({
        errors: {
          ...errors,
          email: `${email} is not a valid email`,
        },
      });

      return false;
    }

    if (isNewUser && newUserType === NewUserType.AdminCreated && !sso_enabled) {
      if (!validatePresence(password)) {
        this.setState({
          errors: {
            ...errors,
            password: "Password field must be completed",
          },
        });

        return false;
      }
      if (!validPassword(password)) {
        this.setState({
          errors: {
            ...errors,
            password: "Password must meet the criteria below",
          },
        });

        return false;
      }
    }

    return true;
  };

  renderGlobalRoleForm = (): JSX.Element => {
    const { onGlobalUserRoleChange } = this;
    const {
      formData: { global_role },
    } = this.state;
    const { isPremiumTier } = this.props;
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
        <p className={`${baseClass}__label`}>Role</p>
        <Dropdown
          value={global_role || "Observer"}
          className={`${baseClass}__global-role-dropdown`}
          options={globalUserRoles}
          searchable={false}
          onChange={onGlobalUserRoleChange}
        />
      </>
    );
  };

  renderNoTeamsMessage = (): JSX.Element => {
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

  renderTeamsForm = (): JSX.Element => {
    const {
      onSelectedTeamChange,
      renderNoTeamsMessage,
      onTeamRoleChange,
    } = this;
    const {
      availableTeams,
      isModifiedByGlobalAdmin,
      defaultTeamRole,
      currentTeam,
    } = this.props;
    const {
      formData: { teams },
    } = this.state;

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
                usersCurrentTeams={teams}
                onFormChange={onSelectedTeamChange}
              />
            </>
          ) : (
            <>
              <p className={`${baseClass}__label`}>Role</p>
              <SelectRoleForm
                currentTeam={currentTeam || teams[0]}
                teams={teams}
                defaultTeamRole={defaultTeamRole || "observer"}
                onFormChange={onTeamRoleChange}
              />
            </>
          ))}
        {!availableTeams.length && renderNoTeamsMessage()}
      </>
    );
  };

  render(): JSX.Element {
    const {
      errors,
      formData: { email, name, newUserType, password, sso_enabled },
      isGlobalUser,
    } = this.state;
    const {
      onCancel,
      submitText,
      isPremiumTier,
      smtpConfigured,
      canUseSso,
      isNewUser,
      currentTeam,
      isModifiedByGlobalAdmin,
      serverErrors,
      availableTeams,
    } = this.props;
    const {
      onFormSubmit,
      onInputChange,
      onCheckboxChange,
      onRadioChange,
      onIsGlobalUserChange,
      renderGlobalRoleForm,
      renderTeamsForm,
    } = this;

    if (!isPremiumTier && !isGlobalUser) {
      console.log(
        `Note: Fleet Free UI does not have teams options.\n
        User ${name} is already assigned to a team and cannot be reassigned without access to Fleet Premium UI.`
      );
    }

    return (
      <form className={baseClass} autoComplete="off">
        {/* {baseError && <div className="form__base-error">{baseError}</div>} */}
        <InputFieldWithIcon
          autofocus
          error={errors.name}
          name="name"
          onChange={onInputChange("name")}
          placeholder="Full name"
          value={name || ""}
        />
        <div
          className="email-disabled"
          data-tip
          data-for="email-disabled-tooltip"
          data-tip-disable={isNewUser || smtpConfigured}
        >
          <InputFieldWithIcon
            error={errors.email || serverErrors?.email}
            name="email"
            onChange={onInputChange("email")}
            placeholder="Email"
            value={email || ""}
            disabled={!isNewUser && !smtpConfigured}
          />
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="email-disabled-tooltip"
          backgroundColor="#3e4771"
          data-html
        >
          <span className={`${baseClass}__tooltip-text`}>
            Editing an email address requires that SMTP is <br />
            configured in order to send a validation email. <br />
            <br />
            Users with Admin role can configure SMTP in
            <br />
            <strong>Settings &gt; Organization settings</strong>.
          </span>
        </ReactTooltip>
        <div className={`${baseClass}__sso-input`}>
          <div
            className="sso-disabled"
            data-tip
            data-for="sso-disabled-tooltip"
            data-tip-disable={canUseSso}
            data-offset="{'top': 25, 'left': 100}"
          >
            <Checkbox
              name="sso_enabled"
              onChange={onCheckboxChange("sso_enabled")}
              value={canUseSso && sso_enabled}
              disabled={!canUseSso}
              wrapperClassName={`${baseClass}__invite-admin`}
            >
              Enable single sign on
            </Checkbox>
            <p className={`${baseClass}__sso-input sublabel`}>
              Password authentication will be disabled for this user.
            </p>
          </div>
          <ReactTooltip
            place="bottom"
            type="dark"
            effect="solid"
            id="sso-disabled-tooltip"
            backgroundColor="#3e4771"
            data-html
          >
            <span className={`${baseClass}__tooltip-text`}>
              Enabling single sign on for a user requires that SSO is <br />
              first enabled for the organization. <br />
              <br />
              Users with Admin role can configure SSO in
              <br />
              <strong>Settings &gt; Organization settings</strong>.
            </span>
          </ReactTooltip>
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
                    checked={newUserType !== NewUserType.AdminInvited}
                    value={NewUserType.AdminCreated}
                    name={"newUserType"}
                    onChange={onRadioChange("newUserType")}
                  />
                  <div
                    className="invite-disabled"
                    data-tip
                    data-for="invite-disabled-tooltip"
                    data-tip-disable={smtpConfigured}
                  >
                    <Radio
                      className={`${baseClass}__radio-input`}
                      label={"Invite user"}
                      id={"invite-user"}
                      disabled={!smtpConfigured}
                      checked={newUserType === NewUserType.AdminInvited}
                      value={NewUserType.AdminInvited}
                      name={"newUserType"}
                      onChange={onRadioChange("newUserType")}
                    />
                    <ReactTooltip
                      place="bottom"
                      type="dark"
                      effect="solid"
                      id="invite-disabled-tooltip"
                      backgroundColor="#3e4771"
                      data-html
                    >
                      <span className={`${baseClass}__tooltip-text`}>
                        The &quot;Invite user&quot; feature requires that SMTP
                        is
                        <br />
                        configured in order to send invitation emails. <br />
                        <br />
                        SMTP can be configured in{" "}
                        <strong>
                          Settings &gt; <br />
                          Organization settings
                        </strong>
                        .
                      </span>
                    </ReactTooltip>
                  </div>
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
            {newUserType !== NewUserType.AdminInvited && !sso_enabled && (
              <>
                <div className={`${baseClass}__password`}>
                  <InputField
                    error={errors.password}
                    name="password"
                    onChange={onInputChange("password")}
                    placeholder="Password"
                    value={password || ""}
                    type="password"
                    hint={[
                      "Must include 7 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)",
                    ]}
                  />
                </div>
                <div className={`${baseClass}__details`}>
                  <IconToolTip
                    isHtml
                    text={`\
                      <div class="password-tooltip-text">\
                        <p>This password is temporary. This user will be asked to set a new password after logging in to the Fleet UI.</p>\
                        <p>This user will not be asked to set a new password after logging in to fleetctl or the Fleet API.</p>\
                      </div>\
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

        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
          >
            {submitText}
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    );
  }
}

export default UserForm;
