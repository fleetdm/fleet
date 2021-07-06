import React, { Component, FormEvent } from "react";
import ReactTooltip from "react-tooltip";

import { ITeam } from "interfaces/team";
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";
import validEmail from "components/forms/validators/valid_email";

// ignore TS error for now until these are rewritten in ts.
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

const baseClass = "create-user-form";

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
  sso_enabled: boolean;
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
  canUseSSO?: boolean;
  defaultName?: string;
  defaultEmail?: string;
  currentUserId?: number;
  defaultGlobalRole?: string | null;
  defaultTeams?: ITeam[];
  isBasicTier: boolean;
  validationErrors: any[]; // TODO: proper interface for validationErrors
  smtpConfigured: boolean;
}

interface ICreateUserFormState {
  errors: {
    email: string | null;
    name: string | null;
    sso_enabled: boolean | null;
  };
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
        sso_enabled: null,
      },
      formData: {
        email: props.defaultEmail || "",
        name: props.defaultName || "",
        sso_enabled: props.canUseSSO || false,
        global_role: props.defaultGlobalRole || null,
        teams: props.defaultTeams || [],
        currentUserId: props.currentUserId,
      },
      isGlobalUser: props.defaultGlobalRole !== null,
    };

    const { isBasicTier } = props;
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

  onFormSubmit = (evt: FormEvent): void => {
    const { createSubmitData, validate } = this;
    evt.preventDefault();
    const valid = validate();
    if (valid) {
      const { onSubmit } = this.props;
      return onSubmit(createSubmitData());
    }
  };

  createSubmitData = (): IFormData => {
    const { currentUserId } = this.props;
    const {
      isGlobalUser,
      formData: { email, name, sso_enabled, global_role, teams },
    } = this.state;

    const submitData = {
      email,
      name,
      sso_enabled,
      currentUserId,
    };
    return isGlobalUser
      ? { ...submitData, global_role, teams: [] }
      : { ...submitData, global_role: null, teams };
  };

  validate = (): boolean => {
    const {
      errors,
      formData: { email },
    } = this.state;

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

    return true;
  };

  renderGlobalRoleForm = (): JSX.Element => {
    const { onGlobalUserRoleChange } = this;
    const {
      formData: { global_role },
    } = this.state;
    const { isBasicTier } = this.props;
    return (
      <>
        {isBasicTier && (
          <InfoBanner className={`${baseClass}__user-permissions-info`}>
            <p>
              Global users can only be members of the top level team and can
              manage or observe all users, entities, and settings in Fleet.
            </p>
            <a
              href="https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/9-Permissions.md#permissions"
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

  renderTeamsForm = (): JSX.Element => {
    const { onSelectedTeamChange } = this;
    const { availableTeams, isBasicTier } = this.props;
    const {
      formData: { teams },
    } = this.state;

    return (
      <>
        <InfoBanner className={`${baseClass}__user-permissions-info`}>
          <p>
            Users can be members of multiple teams and can only manage or
            observe team-sepcific users, entities, and settings in Fleet.
          </p>
          <a
            href="https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/9-Permissions.md#team-member-permissions"
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
    );
  };

  render(): JSX.Element {
    const {
      errors,
      formData: { email, name, sso_enabled },
      isGlobalUser,
    } = this.state;
    const { onCancel, submitText, isBasicTier, smtpConfigured } = this.props;
    const {
      onFormSubmit,
      onInputChange,
      onCheckboxChange,
      onIsGlobalUserChange,
      renderGlobalRoleForm,
      renderTeamsForm,
    } = this;

    if (!isBasicTier && !isGlobalUser) {
      console.log(
        `Note: Fleet Core UI does not have teams options.\n
        User ${name} is already assigned to a team and cannot be reassigned without access to Fleet Basic UI.`
      );
    }

    return (
      <form className={baseClass}>
        {/* {baseError && <div className="form__base-error">{baseError}</div>} */}
        <InputFieldWithIcon
          autofocus
          error={errors.name}
          name="name"
          onChange={onInputChange("name")}
          placeholder="Full name"
          value={name}
        />
        <div
          className="smtp-not-configured"
          data-tip
          data-for="smtp-tooltip"
          data-tip-disable={smtpConfigured}
        >
          <InputFieldWithIcon
            error={errors.email}
            name="email"
            onChange={onInputChange("email")}
            placeholder="Email"
            value={email}
            disabled={!smtpConfigured}
          />
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="smtp-tooltip"
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
          <Checkbox
            name="sso_enabled"
            onChange={onCheckboxChange("sso_enabled")}
            value={sso_enabled}
            disabled={!this.props.canUseSSO}
            wrapperClassName={`${baseClass}__invite-admin`}
          >
            Enable single sign on
          </Checkbox>
          <p className={`${baseClass}__sso-input sublabel`}>
            Password authentication will be disabled for this user.
          </p>
        </div>
        {isBasicTier && (
          <div className={`${baseClass}__selected-teams-container`}>
            <div className={`${baseClass}__team-radios`}>
              <p className={`${baseClass}__label`}>Team</p>
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
              />
            </div>
            <div className={`${baseClass}__teams-form-container`}>
              {isGlobalUser ? renderGlobalRoleForm() : renderTeamsForm()}
            </div>
          </div>
        )}
        {!isBasicTier && renderGlobalRoleForm()}

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
